package gortree

func (t *RTree) Entries() []Spatial {

	var entries []Spatial

	stack := NewStackFrom(t.root)

	for !stack.Empty() {

		cur, _ := stack.Pop()

		if cur.IsLeaf {
			for _, e := range cur.Children {
				entries = append(entries, e.Data)
			}
		} else {
			stack.Push(cur.Children...)
		}

	}

	return entries
}

// Query finds all items intersecting the given Rect
func (t *RTree) Query(r Rect) []Spatial {

	stack := NewStackFrom(t.root)
	var results []Spatial

	for !stack.Empty() {

		cur, _ := stack.Pop()

		// Skip non-intersecting branches
		if !cur.BoundingBox.Intersects(r) {
			continue
		}

		// We have a leaf, return all intersecting entries
		if cur.IsLeaf {
			for _, e := range cur.Children {
				if e.BoundingBox.Intersects(r) {
					results = append(results, e.Data)
				}
			}
		} else {
			// We have an internal node. Add all children to be processed
			stack.Push(cur.Children...)
		}
	}

	return results
}
