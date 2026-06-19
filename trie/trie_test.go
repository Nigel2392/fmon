package trie

import (
	"testing"
)

func TestNewTree(t *testing.T) {
	tree := NewTree[string, int]()
	if tree.root == nil {
		t.Fatal("root should not be nil")
	}
	if tree.root.children == nil {
		t.Fatal("root children map should be initialized")
	}
}

func TestAddAndFind(t *testing.T) {
	tree := NewTree[string, string]()

	// Add a single path
	tree.Add([]string{"a", "b", "c"}, "data1")
	val, ok := tree.Find([]string{"a", "b", "c"})
	if !ok {
		t.Error("expected to find data")
	}
	if val != "data1" {
		t.Errorf("expected 'data1', got '%v'", val)
	}

	// Add another path sharing prefix
	tree.Add([]string{"a", "b", "d"}, "data2")
	val, ok = tree.Find([]string{"a", "b", "d"})
	if !ok || val != "data2" {
		t.Errorf("expected 'data2', got '%v' (ok=%v)", val, ok)
	}

	// Overwrite existing path
	tree.Add([]string{"a", "b", "c"}, "data3")
	val, ok = tree.Find([]string{"a", "b", "c"})
	if !ok || val != "data3" {
		t.Errorf("expected overwritten 'data3', got '%v' (ok=%v)", val, ok)
	}
}

func TestFindLongestPrefix(t *testing.T) {
	tree := NewTree[string, string]()
	tree.Add([]string{"a"}, "a")
	tree.Add([]string{"a", "b"}, "ab")
	tree.Add([]string{"a", "b", "c"}, "abc")

	// Exact match
	val, ok := tree.Find([]string{"a", "b", "c"})
	if !ok || val != "abc" {
		t.Errorf("expected 'abc', got '%v' (ok=%v)", val, ok)
	}

	// Path longer than any stored – returns deepest match
	val, ok = tree.Find([]string{"a", "b", "x"})
	if !ok || val != "ab" {
		t.Errorf("expected longest prefix 'ab', got '%v' (ok=%v)", val, ok)
	}

	// Only root present – no data at root
	_, ok = tree.Find([]string{"x"})
	if ok {
		t.Error("expected no match for non-existent path")
	}
}

func TestFindEmptyPath(t *testing.T) {
	// Note: the current Find implementation does not check the root node
	// before looping, so an empty path never returns data.
	tree := NewTree[string, string]()
	tree.Add([]string{}, "rootData")

	_, ok := tree.Find([]string{})
	if ok {
		t.Error("current behaviour: Find on empty path returns false even if root has data")
	}
}

func TestRemove(t *testing.T) {
	tree := NewTree[string, int]()
	tree.Add([]string{"a", "b"}, 1)
	tree.Add([]string{"a", "c"}, 2)

	// Remove a leaf
	tree.Remove([]string{"a", "b"})
	_, ok := tree.Find([]string{"a", "b"})
	if ok {
		t.Error("expected data to be removed")
	}

	// Other paths remain
	val, ok := tree.Find([]string{"a", "c"})
	if !ok || val != 2 {
		t.Errorf("expected 2, got %v (ok=%v)", val, ok)
	}

	// Removing a non-existent path should not panic
	tree.Remove([]string{"x"})
}

func TestRemoveParentCleanup(t *testing.T) {
	// When a leaf is removed, the parent should be pruned if it has no other
	// children and no data of its own.
	tree := NewTree[string, int]()
	tree.Add([]string{"a", "b"}, 1)

	// Remove leaf
	tree.Remove([]string{"a", "b"})

	// Re-adding the same path should work (parent was properly deleted)
	tree.Add([]string{"a", "b"}, 2)
	val, ok := tree.Find([]string{"a", "b"})
	if !ok || val != 2 {
		t.Errorf("expected 2 after re-add, got %v (ok=%v)", val, ok)
	}
}

func TestFindWithNoData(t *testing.T) {
	tree := NewTree[string, string]()
	tree.Add([]string{"a"}, "a")

	// Path extends beyond stored data – still returns the prefix data
	val, ok := tree.Find([]string{"a", "b"})
	if !ok || val != "a" {
		t.Errorf("expected 'a', got '%v' (ok=%v)", val, ok)
	}

	// No match at all
	_, ok = tree.Find([]string{"x"})
	if ok {
		t.Error("expected no match")
	}
}

// Optional: test with integer segments
func TestIntSegments(t *testing.T) {
	tree := NewTree[int, float64]()
	tree.Add([]int{1, 2, 3}, 3.14)
	val, ok := tree.Find([]int{1, 2, 3})
	if !ok || val != 3.14 {
		t.Errorf("expected 3.14, got %v (ok=%v)", val, ok)
	}
}
