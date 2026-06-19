package watcher

import (
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Nigel2392/fmon/trie"
	"github.com/Nigel2392/fmon/watcher/configure"
	"github.com/bep/debounce"
	"github.com/elliotchance/orderedmap/v2"
)

type WatchedNode struct {
	Path     string
	Object   *configure.MonitoredObject
	Actions  map[configure.ActionType][]*configure.MonitoredObjectAction
	Debounce map[string]func(f func()) // map of action ID to debounce func
}

type WatchTree struct {
	*trie.Tree[string, *WatchedNode]
	keys *orderedmap.OrderedMap[string, struct{}]
	mu   sync.RWMutex
	w    *Watcher
}

func NewWatchTree(w *Watcher) *WatchTree {
	return &WatchTree{
		Tree: trie.NewTree[string, *WatchedNode](),
		keys: orderedmap.NewOrderedMap[string, struct{}](),
		mu:   sync.RWMutex{},
		w:    w,
	}
}

func (w *WatchTree) Keys() []string {
	return w.keys.Keys()
}

func (w *WatchTree) Add(path string, obj *configure.MonitoredObject) {
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")

	w.mu.Lock()
	defer w.mu.Unlock()

	var watched = &WatchedNode{
		Path:     path,
		Object:   obj,
		Actions:  make(map[configure.ActionType][]*configure.MonitoredObjectAction),
		Debounce: make(map[string]func(f func())),
	}

	for _, action := range obj.Actions {
		watched.Actions[action.ActionType] = append(
			watched.Actions[action.ActionType],
			&action,
		)

		if action.Debounce < 0.1 {
			w.w.log.Errorf(
				"Debounce time is too low for action %q in path %q",
				action.ID, path,
			)
			continue
		}

		watched.Debounce[action.ID] = debounce.New(
			time.Duration(action.Debounce * float64(time.Second)),
		)
	}

	w.Tree.Add(parts, watched)
	w.keys.Set(path, struct{}{})
}

func (w *WatchTree) Find(path string) (*WatchedNode, bool) {
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.Tree.Find(parts)
}

func (w *WatchTree) Remove(path string) (removed bool) {
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.Tree.Remove(parts) && w.keys.Delete(path)
}
