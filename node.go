package gortree

type node struct {
	BoundingBox Rect
	IsLeaf      bool
	Children    []*node
	Parent      *node
	Data        Spatial
}

// newLeafNode creates an entry node with data.
func newLeafNode(data Spatial) *node {

	newEntry := &node{
		Data:        data,
		BoundingBox: data.BoundingBox(),
	}

	return newEntry
}
