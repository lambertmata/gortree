package rtree

type Node struct {
	BoundingBox Rect
	IsLeaf      bool
	Entries     []*Entry
	Children    []*Node
	Parent      *Node
}
