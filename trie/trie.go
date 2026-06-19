package trie

type trieNode[Segment comparable, Data any] struct {
	children map[Segment]*trieNode[Segment, Data]

	// keep as pointer to data
	// this might introduce some extra pointer indirection
	// but saves memory over storing a bool
	data *Data
}

type Tree[Segment comparable, Data any] struct {
	root *trieNode[Segment, Data]
}

func NewTree[Segment comparable, Data any]() *Tree[Segment, Data] {
	return &Tree[Segment, Data]{
		root: &trieNode[Segment, Data]{
			children: make(map[Segment]*trieNode[Segment, Data]),
		},
	}
}

func (pt *Tree[Segment, Data]) Add(path []Segment, data Data) {
	current := pt.root
	for _, segment := range path {
		// Add new node to children if none exists
		if _, exists := current.children[segment]; !exists {
			current.children[segment] = &trieNode[Segment, Data]{
				children: make(map[Segment]*trieNode[Segment, Data]),
			}
		}

		// Move down the tree
		current = current.children[segment]
	}

	current.data = &data
}

func (pt *Tree[Segment, Data]) Find(path []Segment) (Data, bool) {
	current := pt.root
	var lastValidData *Data

	for _, segment := range path {
		next, exists := current.children[segment]
		if !exists {
			// Path diverged from our tree. Stop searching.
			break
		}
		current = next

		// If we pass through a node that has explicit data, remember it.
		// We care about the deepest point in the tree.
		if current.data != nil {
			lastValidData = current.data
		}
	}

	if lastValidData != nil {
		return *lastValidData, true
	}

	return *new(Data), false
}

func (pt *Tree[Segment, Data]) Remove(path []Segment) bool {
	r, _ := pt.remove(pt.root, path, 0)
	return r
}

func (pt *Tree[Segment, Data]) remove(current *trieNode[Segment, Data], path []Segment, index int) (removed bool, removeParent bool) {
	if current == nil {
		return false, false
	}

	// we have reached target node
	if index == len(path) {
		current.data = nil

		// when no children are left, let the parent prune it.
		return true, len(current.children) == 0
	}

	segment := path[index]
	child, exists := current.children[segment]
	if !exists {
		return false, false
	}

	// remove nodes bottom up.
	// if true, the child holds no other references
	// and can be deleted.
	if removed, delParent := pt.remove(child, path, index+1); delParent {
		delete(current.children, segment)

		// parent of *this* node can prune *this* node
		// when no data or children are present.
		return removed, current.data == nil && len(current.children) == 0
	}

	return false, false
}
