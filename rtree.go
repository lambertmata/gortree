package rtree

import (
	"fmt"
	"math"
	"slices"
)

type GeoReferenced interface {
	BoundingBox() Rect
	ID() string
}

type RTree struct {
	root       *Node
	maxEntries int
	minEntries int
}

const (
	MinEntries = 2
	MaxEntries = 4
)

func NewRTree() *RTree {
	return &RTree{
		maxEntries: MaxEntries,
		minEntries: MinEntries,
		root: &Node{
			IsLeaf: true,
		},
	}
}

// validateParams checks the min and max entries parameters for an R-tree.
func validateMinMax(min, max int) error {
	if min < MinEntries || max < 2*min {
		return fmt.Errorf("min=%d, max=%d (must satisfy 2 ≤ min ≤ max/2)", min, max)
	}
	return nil
}

// NewRTreeWithMinMax create r-tree with min and max entries parameters.
func NewRTreeWithMinMax(min, max int) (*RTree, error) {
	rt := NewRTree()

	if err := validateMinMax(min, max); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	rt.minEntries = min
	rt.maxEntries = max
	return rt, nil
}

// Min the min entries for each node.
func (t *RTree) Min() int {
	return t.minEntries
}

// Max the max entries for each node.
func (t *RTree) Max() int {
	return t.maxEntries
}

// chooseLeaf selects the best node for inserting a new entry.
func (t *RTree) chooseLeaf(node *Node, entry *Entry) *Node {

	// The tree is descended until a leaf is reached, by selecting the child node which requires the least enlargement
	// to contain rect.
	// If a tie occurs, meaning two child have the same enlargement, the node with the smallest area is selected.
	if node.IsLeaf {
		return node
	}

	var bestNode *Node
	minEnlargement := math.MaxFloat64

	for _, child := range node.Children {

		if node.IsLeaf {
			continue
		}

		enlargement := child.BoundingBox.Enlargement(entry.BoundingBox)

		if enlargement < minEnlargement {

			minEnlargement = enlargement
			bestNode = child

		} else if enlargement == minEnlargement && bestNode != nil && (child.BoundingBox.Area() < bestNode.BoundingBox.Area()) {
			bestNode = child
		}
	}

	// If no best node found, return current node
	if bestNode == nil {
		return node
	}

	return t.chooseLeaf(bestNode, entry)
}

// updateNodeMBR Using current entries MBRs it updated the node BoundingBox.
func (t *RTree) updateNodeMBR(node *Node) {
	if node.IsLeaf {
		node.BoundingBox = computeEntriesMBR(node.Entries)
	} else {
		node.BoundingBox = computeNodesMBR(node.Children)
	}
}

// updateMBRsUpward updates MBRs starting from node up to the root.
func (t *RTree) updateMBRsUpward(node *Node) {
	for node != nil {
		t.updateNodeMBR(node)
		node = node.Parent
	}
}

// adjustTree updates the MBRs up the tree after an insertion
func (t *RTree) adjustTree(node *Node, splitNode *Node) {

	// Case 1: If no split occurred, just update MBRs up the tree
	if splitNode == nil {
		t.updateMBRsUpward(node)
		return
	}

	// Case 2: Root split
	if node.Parent == nil {

		// Create a new root
		newRoot := &Node{
			IsLeaf:   false,
			Children: []*Node{node, splitNode},
		}

		// Update parent references
		node.Parent = newRoot
		splitNode.Parent = newRoot

		// Update tree's root
		t.root = newRoot

		// Update the BoundingBox of the new root
		newRoot.BoundingBox = computeNodesMBR(newRoot.Children)

		return
	}

	// Case 3: Split occurred at non-root level

	// We need to add the new node to the parent and continue adjusting upward
	parent := node.Parent

	// Update the BoundingBox of the original node
	node.BoundingBox = computeNodesMBR(node.Children)

	// Add splitNode to parent
	parent.Children = append(parent.Children, splitNode)
	splitNode.Parent = parent

	// Check if parent needs splitting
	parentSplit := t.splitNodeIfNeeded(parent)

	// Continue adjusting up the tree
	t.adjustTree(parent, parentSplit)

}

// Insert adds a new item to the tree.
func (t *RTree) Insert(data GeoReferenced) {

	// Create the new entry node
	newEntry := NewEntry(data)

	// Find the best leaf node to insert the new entry node.
	leaf := t.chooseLeaf(t.root, newEntry)

	// Add entry node to leaf
	leaf.Entries = append(leaf.Entries, newEntry)
	newEntry.Parent = leaf

	// Split if the leaf overflows
	splitNode := t.splitNodeIfNeeded(leaf)

	// propagate changes upward
	t.adjustTree(leaf, splitNode)

}

// pickSeeds gives the two entries that are the farthest apart
func (t *RTree) pickSeeds(nodeA *Node) [2]*Entry {

	seeds := [2]*Entry{}

	var maxEnlargement float64

	// Pick the entries that (would waste the more area if put together).
	for i := 0; i < len(nodeA.Entries); i++ {
		for j := 0; j < len(nodeA.Entries); j++ {
			enlargement := nodeA.Entries[i].BoundingBox.Enlargement(nodeA.Entries[j].BoundingBox)
			if enlargement > maxEnlargement {
				maxEnlargement = enlargement
				seeds[0] = nodeA.Entries[i]
				seeds[1] = nodeA.Entries[j]
			}
		}
	}

	return seeds
}

func computeEntriesMBR(entries []*Entry) Rect {
	var mbr Rect
	for i := 0; i < len(entries); i++ {
		mbr.Expand(entries[i].BoundingBox)
	}
	return mbr
}

func computeNodesMBR(nodes []*Node) Rect {
	var mbr Rect
	for i := 0; i < len(nodes); i++ {
		mbr.Expand(nodes[i].BoundingBox)
	}
	return mbr
}

func (t *RTree) splitNodeIfNeeded(node *Node) *Node {
	if len(node.Entries) <= t.maxEntries {
		return nil
	}
	return t.splitNode(node)
}

// splitNode performs quadratic split.
func (t *RTree) splitNode(node *Node) *Node {

	// Pick two entries that are furthest apart
	seeds := t.pickSeeds(node)

	// Create two nodes and assign a seed each.
	// groupA node replaces the current node.
	// groupB node is a new node and will be assigned part of the entries of the original node.
	groupA := &Node{
		BoundingBox: seeds[0].BoundingBox,
		Entries:     []*Entry{seeds[0]},
		IsLeaf:      node.IsLeaf,
		Parent:      node.Parent,
	}

	groupB := &Node{
		BoundingBox: seeds[1].BoundingBox,
		Entries:     []*Entry{seeds[1]},
		IsLeaf:      node.IsLeaf,
		Parent:      node.Parent,
	}

	// Collect the remaining entries that aren't seeds to distribute
	remaining := slices.DeleteFunc(node.Entries, func(entry *Entry) bool {
		return entry == seeds[0] || entry == seeds[1]
	})

	// Distribute remaining entries between groupA and groupB
	for len(remaining) > 0 {

		next := t.pickNext(remaining, groupA, groupB)
		entry := remaining[next]
		remaining = append(remaining[:next], remaining[next+1:]...)

		targetNode := t.chooseGroup(entry, groupA, groupB)

		targetNode.Entries = append(targetNode.Entries, entry)
		targetNode.BoundingBox.Expand(entry.BoundingBox)
	}

	// Replace original node with groupA
	*node = *groupA

	// This is a critical part. We need to make sure that previous groupA entries point to node.
	t.adjustEntriesParent(node)
	t.adjustEntriesParent(groupB)

	// Return groupB as the new split node
	return groupB
}

// chooseGroup returns the group where entry should be assigned to.
func (t *RTree) chooseGroup(entry *Entry, groupA, groupB *Node) *Node {

	// Ensure minimum number of entries is met
	if len(groupA.Entries) < t.minEntries {
		return groupA
	}
	if len(groupB.Entries) < t.minEntries {
		return groupB
	}

	// Now choose the one which requires the lease enlargement
	// If it's a tie, chose the one with smallest area
	enlargeA := groupA.BoundingBox.Enlargement(entry.BoundingBox)
	enlargeB := groupB.BoundingBox.Enlargement(entry.BoundingBox)

	if enlargeA < enlargeB {
		return groupA
	}

	if enlargeB < enlargeA {
		return groupB
	}

	if groupA.BoundingBox.Area() < groupB.BoundingBox.Area() {
		return groupA
	}

	return groupB
}

// pickNext returns the index of the entry with the greatest preference to be inserted in a group.
func (t *RTree) pickNext(entries []*Entry, groupA *Node, groupB *Node) int {

	next := 0
	maxDiff := -1.0

	for i, entry := range entries {
		d1 := groupA.BoundingBox.Enlargement(entry.BoundingBox)
		d2 := groupB.BoundingBox.Enlargement(entry.BoundingBox)
		diff := math.Abs(d1 - d2)
		if diff > maxDiff {
			maxDiff = diff
			next = i
		}
	}

	return next
}

// adjustEntriesParent updates the node entries such that their Parent pointer points to the node.
func (t *RTree) adjustEntriesParent(node *Node) {
	for _, child := range node.Entries {
		child.Parent = node
	}
}
