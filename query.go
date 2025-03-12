package gortree

// Query finds all items intersecting the given Rect
func (t *RTree) Query(r Rect) []Spatial {
	if t.root == nil {
		return nil
	}

	var results []Spatial

	t.search(t.root, r, &results)

	return results
}

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

// search performs the recursive search
func (t *RTree) search(n *node, searchRect Rect, results *[]Spatial) {

	if !n.BoundingBox.Intersects(searchRect) {
		return
	}

	for _, cur := range n.Children {

		if cur.IsLeaf {

			// This is a data node
			for _, entry := range cur.Children {
				if entry.BoundingBox.Intersects(searchRect) {
					*results = append(*results, entry.Data)
				}
			}

		} else {
			// This is an internal node
			t.search(cur, searchRect, results)
		}
	}
}
