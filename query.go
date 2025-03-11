package rtree

// Query finds all items intersecting the given Rect
func (t *RTree) Query(searchRect Rect) []GeoReferenced {
	if t.root == nil {
		return nil
	}

	var results []GeoReferenced
	t.search(t.root, searchRect, &results)
	return results
}

func (t *RTree) Entries() []*Entry {

	var entries []*Entry

	stack := []*Node{t.root}

	for len(stack) > 0 {

		curNode := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if curNode.IsLeaf {
			entries = append(entries, curNode.Entries...)
		} else {
			stack = append(stack, curNode.Children...)
		}

	}

	return entries
}

// search performs the recursive search
func (t *RTree) search(node *Node, searchRect Rect, results *[]GeoReferenced) {
	if !node.BoundingBox.Intersects(&searchRect) {
		return
	}

	for _, curNode := range node.Children {

		if curNode.IsLeaf {

			// This is a data node
			for _, entry := range curNode.Entries {
				if entry.BoundingBox.Intersects(&searchRect) {
					*results = append(*results, entry.Data)
				}
			}

		} else {
			// This is an internal node
			t.search(curNode, searchRect, results)
		}
	}
}
