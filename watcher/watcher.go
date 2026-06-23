package watcher

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/dop251/goja"
	"github.com/fsnotify/fsnotify"
	"github.com/kardianos/service"
	"github.com/robfig/cron"
)

var (
	_      service.Interface = (*Watcher)(nil)
	_      cron.Job          = (*watchCron)(nil)
	opsMap                   = map[fsnotify.Op]configure.ActionType{
		// fsnotify.Write:  ,
		fsnotify.Create: configure.CREATE_ACTION,
		fsnotify.Remove: configure.DELETE_ACTION,
		fsnotify.Rename: configure.RENAME_ACTION,
		fsnotify.Write:  configure.CHANGE_ACTION,
		fsnotify.Chmod:  configure.CHANGE_ACTION,
	}
)

type payload struct {
	obj    *WatchedNode
	action *configure.MonitoredObjectAction
	event  fsnotify.Event
}

type watchCron struct {
	watcher *Watcher
	node    *WatchedNode
	action  configure.MonitoredObjectAction
}

func (w *watchCron) Run() {
	w.watcher.queue <- payload{
		obj:    w.node,
		action: &w.action,
	}
}

type Watcher struct {
	confDir   string
	watcher   *fsnotify.Watcher
	watchList map[string]struct{}
	config    *configure.FilesystemMonitor
	nodes     *WatchTree
	done      chan struct{}
	queue     chan payload
	log       service.Logger
	cron      *cron.Cron
}

func NewWatcher(confDir string) *Watcher {
	return &Watcher{
		confDir: confDir,
		done:    make(chan struct{}),
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
	w.queue = make(chan payload)
	w.watchList = make(map[string]struct{})
	w.cron = cron.New()

	if service.Interactive() {
		w.log.Infof("Running in DEBUG / INTERACTIVE mode. Press Ctrl+C to stop.")
	} else {
		w.log.Info("Starting FMon service...")
	}

	if err := w.reloadConfig(s); err != nil {
		return err
	}

	go w.watch(s)
	go w.work(s)
	go w.cron.Run()
	return nil
}

// Stop provides a place to clean up program execution before it is terminated.
// It should not take more then a few seconds to execute.
// Stop should not call os.Exit directly in the function.
func (w *Watcher) Stop(s service.Service) error {
	if w.done != nil {
		close(w.done)
	}

	if w.queue != nil {
		close(w.queue)
	}

	w.cron.Stop()

	if err := w.watcher.Close(); err != nil {
		return err
	}
	w.watcher = nil
	w.nodes = nil
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

// work executes actions in a separate goroutine. It is started by the watcher.
func (w *Watcher) work(s service.Service) {
	var vm = goja.New()

	if err := vm.Set("console", map[string]interface{}{
		"debug":  w.log.Info,
		"debugf": w.log.Infof,
		"info":   w.log.Info,
		"infof":  w.log.Infof,
		"warn":   w.log.Warning,
		"warnf":  w.log.Warningf,
		"error":  w.log.Error,
		"errorf": w.log.Errorf,
	}); err != nil {
		w.log.Errorf("Failed to set up VM console, stopping work: %v", err)
		return
	}

	if err := vm.Set("fsnotify", map[string]interface{}{
		"Create": fsnotify.Create,
		"Write":  fsnotify.Write,
		"Remove": fsnotify.Remove,
		"Rename": fsnotify.Rename,
		"Chmod":  fsnotify.Chmod,
	}); err != nil {
		w.log.Errorf("Failed to set up VM console, stopping work: %v", err)
		return
	}

	if err := vm.Set("filepath", map[string]interface{}{
		"Abs":  filepath.Abs,
		"Base": filepath.Base,
		"Dir":  filepath.Dir,
		"Ext":  filepath.Ext,
		"Join": filepath.Join,
		"Rel":  filepath.Rel,
	}); err != nil {
		w.log.Errorf("Failed to set up VM console, stopping work: %v", err)
		return
	}

	if err := vm.Set("shell", w.execShellCommand); err != nil {
		w.log.Errorf("Failed to set up VM console, stopping work: %v", err)
		return
	}

	for {
		// try to keep indentation at a minimum.
		var (
			p  payload
			ok bool
		)

		select {
		case p, ok = <-w.queue:
		case <-w.done:
			// Stop was called, break out.
			return
		}

		if !ok {
			// queue channel closed
			return
		}

		var actionPath = strings.ToLower(p.action.Action)
		switch {
		case strings.HasSuffix(actionPath, ".js"):
			/*

				JAVASCRIPT FILE

			*/
			exec, ok := p.obj.compiled[p.action.ID]

			// Pre-compile the action for future runs
			if !ok || exec == nil {
				data, err := os.ReadFile(p.action.Action)
				if err != nil {
					w.log.Errorf("Failed to read action file %q: %v", p.action.ID, err)
					continue
				}

				compiled, err := goja.Compile(p.action.ID, string(data), false)
				if err != nil {
					w.log.Errorf("Failed to compile action file JS %q: %v", p.action.ID, err)
					continue
				}

				vmFn, err := vm.RunProgram(compiled)
				if err != nil {
					w.log.Errorf("Error whilst running JS program for %q: %v", p.action.ID, err)
				}

				var fn func(map[string]interface{})
				err = vm.ExportTo(vmFn, &fn)
				if err != nil {
					w.log.Errorf("Script %q did not return a valid function: %v", p.action.ID, err)
					continue
				}

				// Make sure the work goroutine doesn't panic.
				p.obj.compiled[p.action.ID] = func(pld *payload) {
					defer func() {
						if r := recover(); r != nil {
							w.log.Errorf("Script %q panicked: %v", pld.action.ID, r)
						}
					}()

					var ctx = map[string]interface{}{
						"monitored": pld.obj.Path,
						"action":    pld.action,
						"event": map[string]interface{}{
							"Name": pld.event.Name,
							"Op":   pld.event.Op.String(),
						},
					}

					fn(ctx)
				}
			}

			p.obj.compiled[p.action.ID](&p)

			w.log.Infof("Javascript action %q executed successfully.", p.action.ID)

		case strings.HasSuffix(actionPath, ".sh"),
			strings.HasSuffix(actionPath, ".ps1"),
			strings.HasSuffix(actionPath, ".bat"),
			strings.HasSuffix(actionPath, ".cmd"):
			/*

				SHELL FILE

			*/

			var (
				cmd  string
				args = make([]string, 0, 5)
			)

			// platform specific shell execution
			switch filepath.Ext(actionPath) {
			case ".sh":
				cmd = "sh"
				args = append(args, p.action.Action)

			case ".ps1":
				cmd = "powershell.exe"
				args = append(args, "-ExecutionPolicy", "Bypass", "-File", p.action.Action)

			case ".bat", ".cmd":
				cmd = "cmd.exe"
				args = append(args, "/C", p.action.Action)
			}

			args = append(
				args,
				p.obj.Path,
				p.action.ID,
				p.event.Name,
				p.event.Op.String(),
			)

			var command = exec.Command(
				cmd, args...,
			)

			hideCommandWindow(command)

			output, err := command.CombinedOutput()
			if err != nil {
				w.log.Errorf("Shell script failed %q: %v. Output: %s", p.action.ID, err, string(output))
				continue
			}

			w.log.Infof("Shell script %q executed successfully.", p.action.ID)
		}
	}
}

// Handle events from watched directories/files.
func (w *Watcher) event(s service.Service, event fsnotify.Event) error {

	event.Name = filepath.ToSlash(event.Name)

	// Config file has changed, reload the config into memory.
	if strings.HasPrefix(event.Name, filepath.ToSlash(w.confDir)) {
		w.log.Infof("fsnotify event: %s", event.String())
		return w.reloadConfig(s)
	}

	if service.Interactive() {
		w.log.Infof("event: %s", event.String())
	}

	watched, ok := w.nodes.Find(event.Name)
	if ok && event.Has(fsnotify.Create) && watched.Object.Recursive {
		info, err := os.Stat(event.Name)
		if err == nil && info.IsDir() {
			w.log.Infof("New directory detected, adding to watcher: %s", event.Name)
			if err := w.watcher.Add(event.Name); err != nil {
				w.log.Errorf("Failed to watch new directory %q: %v", event.Name, err)
			} else {
				watched.subDirs[event.Name] = struct{}{}
			}
		}
	}

	for fsnotifyOp, configureAction := range opsMap {

		if !event.Has(fsnotifyOp) {
			continue // This operation wasn't monitored, continue...
		}

		watched, ok := w.nodes.Find(event.Name)
		if !ok {
			continue // No paths exist for this operation, continue...
		}

		for _, action := range watched.Actions[configureAction] {
			var debounceObj, ok = watched.Debounce[action.ID]
			if !ok {
				w.log.Warningf("Debounce action not found for action %q in %q", action.ID, event.String())
				continue
			}

			// debounce the action (required)
			// makes sure the system doesn't overload with events
			debounceObj(func() {
				w.queue <- payload{
					obj:    watched,
					event:  event,
					action: action,
				}
			})
		}
	}

	if obj, ok := w.nodes.Find(event.Name); event.Has(fsnotify.Remove) && ok && strings.EqualFold(filepath.ToSlash(obj.Path), event.Name) {
		rem := w.nodes.Remove(event.Name)
		if !rem {
			w.log.Errorf("Failed to remove node %q from watchtree", event.Name)
			return nil
		}
	}

	return nil
}

// Load the config into the service memory.
func (w *Watcher) reloadConfig(s service.Service) (err error) {
	w.config, err = configure.Read()
	if err != nil {
		return err
	}

	w.log.Infof("Initialized configuration file: %q", w.config.Path)

	if w.nodes == nil {
		w.nodes = NewWatchTree(w)
	}

	// cron jobs need to be re-initialized every time
	w.cron.Stop()
	w.cron = cron.New()

	// Remove files that are no longer in the configuration file.
	for _, k := range w.nodes.Keys() {
		if _, ok := w.config.Files.Get(k); !ok {
			var node, ok = w.nodes.Find(k)
			if !ok {
				w.log.Errorf("Path %q not found in watchlist. Removing from watcher.", k)
				continue
			}

			// remove root monitored dir from watcher
			if err = w.watcher.Remove(k); err != nil && err != fsnotify.ErrNonExistentWatch {
				w.log.Errorf("Failed to remove path %q from watchlist: %v", k, err)
			}

			// remove all it's subdirs from the watcher
			for path := range node.subDirs {
				if err = w.watcher.Remove(path); err != nil && err != fsnotify.ErrNonExistentWatch {
					w.log.Errorf("Failed to remove path %q from watchlist: %v", path, err)
				}
			}

			// delete the node from the tree
			if !w.nodes.Remove(k) {
				w.log.Errorf("Failed to remove path %q from memory watchlist", k)
				continue
			}

			w.log.Infof("Removed path %q from watchlist", k)
		}
	}

	// Watch new files that are in the configuration file but were not present before.
	for k, obj := range w.config.Files.Iterator() {
		k = filepath.ToSlash(k)

		var existingSubDirs = make(map[string]struct{})
		node, ok := w.nodes.Find(k)
		if !ok { // add to watcher, add new node object
			err = w.watcher.Add(k)
			if err != nil {
				w.log.Errorf("Failed to add path %q to watchlist: %v", k, err)
				continue
			}

			// add directory paths recursively if requested
			if obj.Recursive {
				err := filepath.WalkDir(k, func(path string, d fs.DirEntry, err error) error {
					if !d.IsDir() {
						return nil
					}

					existingSubDirs[path] = struct{}{}
					return w.watcher.Add(path)
				})

				if err != nil {
					return err
				}
			}

			w.log.Infof("Added path %q to watchlist", k)
		} else {
			existingSubDirs = node.subDirs
		}

		// always update the node to make sure debounce and runtimes match the fresh config
		node = w.nodes.Add(k, obj)
		node.subDirs = existingSubDirs

		for _, action := range obj.Actions {
			if action.Cron == "" {
				continue
			}

			if err := w.cron.AddJob(action.Cron, &watchCron{w, node, action}); err != nil {
				w.log.Errorf("Failed to add cron job for path %q: %v", k, err)
				continue
			}
		}
	}

	return nil
}

func (w *Watcher) execShellCommand(command string) string {
	var (
		cmd  string
		args []string
	)
	switch runtime.GOOS {
	case "windows":
		cmd = "powershell.exe"
		args = []string{"-ExecutionPolicy", "Bypass", "-Command", command}
	default:
		cmd = "sh"
		args = []string{"-c", command}
	}

	var cmdObj = exec.Command(cmd, args...)
	hideCommandWindow(cmdObj)

	output, err := cmdObj.CombinedOutput()
	if err != nil {
		w.log.Errorf("Shell script failed %q: %v. Output: %s", command, err, string(output))
		return ""
	}

	return string(output)
}
