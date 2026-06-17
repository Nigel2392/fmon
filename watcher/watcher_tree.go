package watcher

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/Nigel2392/fmon/trie"
	"github.com/Nigel2392/fmon/watcher/configure"
)

type WatchedNode struct {
	Object *configure.MonitoredObject
}

type WatchTree struct {
	*trie.Tree[string, *WatchedNode]
	mu sync.RWMutex
}

func NewWatchTree() *WatchTree {
	return &WatchTree{
		Tree: trie.NewTree[string, *WatchedNode](),
		mu:   sync.RWMutex{},
	}
}

func (w *WatchTree) Add(path string, obj *WatchedNode) {
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")

	w.mu.Lock()
	defer w.mu.Unlock()

	w.Tree.Add(parts, obj)
}

func (w *WatchTree) Find(path string) (*WatchedNode, bool) {
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")

	w.mu.Lock()
	defer w.mu.Unlock()

	return w.Tree.Find(parts)
}

func (w *WatchTree) Remove(path string, obj *WatchedNode) {
	path = filepath.ToSlash(path)
	parts := strings.Split(path, "/")

	w.mu.Lock()
	defer w.mu.Unlock()

	w.Tree.Remove(parts)
}
