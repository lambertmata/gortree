package gortree

func (t *RTree) Entries() []Spatial {

	var entries []Spatial

	stack := []*node{t.root}

	for len(stack) > 0 {

		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if cur.IsLeaf {
			for _, e := range cur.Children {
				entries = append(entries, e.Data)
			}
		} else {
			stack = append(stack, cur.Children...)
		}

	}

	return entries
}

// Query finds all items intersecting the given Rect
func (t *RTree) Query(r Rect) []Spatial {

	stack := []*node{t.root}
	var results []Spatial

	for len(stack) > 0 {

		lastIdx := len(stack) - 1
		cur := stack[lastIdx]
		stack = stack[:lastIdx]

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
			stack = append(stack, cur.Children...)
		}
	}

	return results
}
