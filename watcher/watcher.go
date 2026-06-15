package watcher

import (
	"github.com/fsnotify/fsnotify"
	"github.com/kardianos/service"
)

var _ service.Interface = (*Watcher)(nil)

type Watcher struct {
	watcher *fsnotify.Watcher
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

	return nil
}

// Stop provides a place to clean up program execution before it is terminated.
// It should not take more then a few seconds to execute.
// Stop should not call os.Exit directly in the function.
func (w *Watcher) Stop(s service.Service) error {
	if err := w.watcher.Close(); err != nil {
		return err
	}
	return nil
}
