package watcher_test

import (
	"sync"
	"testing"

	"github.com/Nigel2392/fmon/watcher"
	"github.com/Nigel2392/fmon/watcher/configure" // assumed to exist
)

func TestNewWatchTree(t *testing.T) {
	wt := watcher.NewWatchTree(watcher.NewWatcher(""))
	if wt == nil {
		t.Fatal("expected non-nil WatchTree")
	}
	// The embedded tree root should be initialised
	if wt.Tree == nil { // Tree is the embedded field (after fixing the import name)
		t.Fatal("embedded trie should not be nil")
	}
}

func TestAddAndFind(t *testing.T) {
	wt := watcher.NewWatchTree(watcher.NewWatcher(""))
	obj := &configure.MonitoredObject{Recursive: true}

	wt.Add("/a/b/c", obj)

	// Exact match
	found, ok := wt.Find("/a/b/c")
	if !ok {
		t.Fatal("expected to find node")
	}
	if found.Object != obj {
		t.Error("found object does not match")
	}

	// Add a shorter prefix
	obj2 := &configure.MonitoredObject{}
	wt.Add("/a/b", obj2)

	// Deepest match should win
	found, ok = wt.Find("/a/b/c")
	if !ok || found.Object != obj {
		t.Error("should return /a/b/c data for exact path")
	}

	// Unknown deeper path returns longest prefix
	found, ok = wt.Find("/a/b/d")
	if !ok || found.Object != obj2 {
		t.Error("should return /a/b data for /a/b/d")
	}

	// Root path
	wt.Add("/", &configure.MonitoredObject{})
	_, ok = wt.Find("/")
	if ok {
		t.Log("Find('/') returned true – root data found")
	}
}

func TestOverrideAddAndFind(t *testing.T) {
	wt := watcher.NewWatchTree(watcher.NewWatcher(""))
	obj := &configure.MonitoredObject{Recursive: true}

	wt.Add("/a/b/c", obj)

	obj2 := &configure.MonitoredObject{Recursive: false}
	wt.Add("/a/b/c", obj2)

	// Exact match
	found, ok := wt.Find("/a/b/c")
	if !ok {
		t.Fatal("expected to find node")
	}
	if found.Object == obj || found.Object != obj2 {
		t.Error("found object does not match")
	}
}

func TestRemove(t *testing.T) {
	wt := watcher.NewWatchTree(watcher.NewWatcher(""))
	wt.Add("/a/b", &configure.MonitoredObject{})

	// Remove (note: the obj argument is not used internally by Remove, but the API accepts it)
	if !wt.Remove("/a/b") {
		t.Error("Failed to remove object /a/b")
	}

	_, ok := wt.Find("/a/b")
	if ok {
		t.Error("expected data to be removed")
	}

	// Remove non-existent path should be safe
	wt.Remove("/x/y")
}

func TestConcurrentAccess(t *testing.T) {
	wt := watcher.NewWatchTree(watcher.NewWatcher(""))
	const goroutines = 50
	var wg sync.WaitGroup

	// Stress test with concurrent adds, finds, and removes
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			path := "/node" + string(rune('a'+id%26))

			wt.Add(path, &configure.MonitoredObject{})
			wt.Find(path)
			wt.Remove(path)
		}(i)
	}
	wg.Wait()
	// No race condition expected
}
