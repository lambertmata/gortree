package rtree

type Node struct {
	BoundingBox Rect
	IsLeaf      bool
	Children    []*Node
	Parent      *Node
	Data        GeoReferenced
}

// NewLeafNode creates an entry Node with data.
func NewLeafNode(data GeoReferenced) *Node {

	newEntry := &Node{
		Data:        data,
		BoundingBox: data.BoundingBox(),
	}

	return newEntry
}
