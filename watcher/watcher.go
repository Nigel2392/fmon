package watcher

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/fsnotify/fsnotify"
	"github.com/kardianos/service"
)

var _ service.Interface = (*Watcher)(nil)

type watcherActionsMeta struct {
	path   string
	obj    *configure.MonitoredObject
	action *configure.MonitoredObjectAction
}

type watcherActions struct {
	m map[configure.ActionType][]*watcherActionsMeta
}

type Watcher struct {
	confDir           string
	watcher           *fsnotify.Watcher
	config            *configure.FilesystemMonitor
	confBuilt         *watcherActions
	currentlyWatching map[string]struct{}
	done              chan struct{}
	log               service.Logger
}

func NewWatcher(confDir string) *Watcher {
	return &Watcher{
		confDir:           confDir,
		currentlyWatching: make(map[string]struct{}),
		done:              make(chan struct{}),
	}
}

// Start provides a place to initiate the service. The service doesn't
// signal a completed start until after this function returns, so the
// Start function must not take more then a few seconds at most.
func (w *Watcher) Start(s service.Service) error {
	if w.watcher != nil {
		w.watcher.Close()
		w.watcher = nil
	}

	var err error
	w.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	err = w.watcher.Add(w.confDir)
	if err != nil {
		return err
	}

	l, err := s.Logger(nil)
	if err != nil {
		return err
	}

	w.log = l
	w.done = make(chan struct{})

	if service.Interactive() {
		w.log.Infof("Running in DEBUG / INTERACTIVE mode. Press Ctrl+C to stop.")
	} else {
		w.log.Info("Starting FMon service...")
	}

	if err := w.reloadConfig(s); err != nil {
		return err
	}

	go w.watch(s)
	return nil
}

// Stop provides a place to clean up program execution before it is terminated.
// It should not take more then a few seconds to execute.
// Stop should not call os.Exit directly in the function.
func (w *Watcher) Stop(s service.Service) error {
	if w.done != nil {
		close(w.done)
	}

	if err := w.watcher.Close(); err != nil {
		return err
	}
	w.watcher = nil
	w.currentlyWatching = make(map[string]struct{})
	return nil
}

// watch will listen to the fsnotify signals, executing actions based
// on constraints.
func (w *Watcher) watch(s service.Service) {
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			err := w.event(s, event)
			if err != nil {
				if errors.Is(err, configure.ErrConfigNotExists) {
					w.log.Errorf("Stopping service: %v", err)
					if err := s.Stop(); err != nil {
						os.Exit(1)
					}
					return
				}
				w.log.Errorf("Error while handling fsnotify event: %v", err)
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}

			w.log.Errorf("Error during watcher execution: %v", err)
		case <-w.done:
			// Stop() was called, break out of the loop
			return
		}
	}
}

// Handle events from watched directories/files.
func (w *Watcher) event(s service.Service, event fsnotify.Event) error {

	// Config file has changed, reload the config into memory.
	if strings.HasPrefix(filepath.ToSlash(event.Name), filepath.ToSlash(w.confDir)) {
		w.log.Infof("fsnotify event: %s", event.String())
		return w.reloadConfig(s)
	}

	if _, ok := w.currentlyWatching[event.Name]; event.Has(fsnotify.Remove) && ok {
		delete(w.currentlyWatching, event.Name)
	}

	var ops = []fsnotify.Op{
		fsnotify.Create,
		fsnotify.Write,
		fsnotify.Remove,
		fsnotify.Rename,
		fsnotify.Chmod,
	}

	for _, op := range ops {
		if !event.Has(op) {
			continue
		}

		// var objs, ok = w.confBuilt.m[op.String()]

	}

	return nil
}

// Load the config into the service memory.
func (w *Watcher) reloadConfig(s service.Service) (err error) {
	w.config, err = configure.Read()
	if err != nil {
		return err
	}

	if err = w.rebuildConfig(); err != nil {
		return err
	}

	w.log.Infof("Initialized configuration file: %q", w.config.Path)

	if w.currentlyWatching == nil {
		w.currentlyWatching = make(map[string]struct{})
	}

	// Remove files that are no longer in the configuration file.
	for k := range w.currentlyWatching {
		if _, ok := w.config.Files.Get(k); !ok && w.confDir != "" {
			if err = w.watcher.Remove(k); err != nil && err != fsnotify.ErrNonExistentWatch {
				w.log.Errorf("Failed to remove path %q from watchlist: %v", k, err)
			}
			w.log.Infof("Removed path %q from watchlist", k)
			delete(w.currentlyWatching, k)
		}
	}

	// Watch new files that are in the configuration file but were not present before.
	for _, k := range w.config.Files.Keys() {
		if _, ok := w.currentlyWatching[k]; !ok && w.confDir != "" {
			err = w.watcher.Add(k)
			if err != nil {
				w.log.Errorf("Failed to add path %q to watchlist: %v", k, err)
				continue
			}
			w.currentlyWatching[k] = struct{}{}
			w.log.Infof("Added path %q to watchlist", k)
		}
	}

	return nil
}

func (w *Watcher) rebuildConfig() error {
	w.confBuilt = &watcherActions{
		m: make(map[configure.ActionType][]*watcherActionsMeta),
	}

	for _, action := range configure.ACTION_TYPES {
		w.confBuilt.m[action] = make([]*watcherActionsMeta, 0)
	}

	for path, obj := range w.config.Files.Iter() {
		for _, action := range obj.Actions {
			var meta = &watcherActionsMeta{
				path:   path,
				obj:    obj,
				action: &action,
			}

			w.confBuilt.m[action.ActionType] = append(
				w.confBuilt.m[action.ActionType], meta,
			)
		}
	}

	return nil
}
