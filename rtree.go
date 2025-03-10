package rtree

import (
	"fmt"
	"math"
	"slices"
)

type GeoReferenced interface {
	Bounds() [4]float64
}

type Node struct {
	MBR     Rect
	IsLeaf  bool
	Entries []*Node
	Parent  *Node
	Data    GeoReferenced
}

type RTree struct {
	root       *Node
	maxEntries int
	minEntries int
}

const MinEntries = 2
const MaxEntries = 4

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
func (t *RTree) chooseLeaf(node *Node, entry *Node) *Node {

	// The tree is descended until a leaf is reached, by selecting the child node which requires the least enlargement
	// to contain rect.
	// If a tie occurs, meaning two child have the same enlargement, the node with the smallest area is selected.
	if node.Data != nil {
		return node
	}

	var bestNode *Node
	minEnlargement := math.MaxFloat64

	for _, child := range node.Entries {

		if child.Data != nil {
			continue
		}

		enlargement := child.MBR.Enlargement(entry.MBR)

		if enlargement < minEnlargement {

			minEnlargement = enlargement
			bestNode = child

		} else if enlargement == minEnlargement && bestNode != nil && (child.MBR.Area() < bestNode.MBR.Area()) {
			bestNode = child
		}
	}

	// If no best node found, return current node
	if bestNode == nil {
		return node
	}

	return t.chooseLeaf(bestNode, entry)
}

// updateMBR Using current entries MBRs it updated the node MBR.
func (t *RTree) updateMBR(node *Node) {
	node.MBR = computeMBR(node.Entries)
}

// updateMBRsUpward updates MBRs starting from node up to the root.
func (t *RTree) updateMBRsUpward(node *Node) {
	for node != nil {
		t.updateMBR(node)
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
			IsLeaf:  false,
			Entries: []*Node{node, splitNode},
		}

		// Update parent references
		node.Parent = newRoot
		splitNode.Parent = newRoot

		// Update tree's root
		t.root = newRoot

		// Update the MBR of the new root
		newRoot.MBR = computeMBR(newRoot.Entries)

		return
	}

	// Case 3: Split occurred at non-root level

	// We need to add the new node to the parent and continue adjusting upward
	parent := node.Parent

	// Update the MBR of the original node
	node.MBR = computeMBR(node.Entries)

	// Add splitNode to parent
	parent.Entries = append(parent.Entries, splitNode)
	splitNode.Parent = parent

	// Check if parent needs splitting
	parentSplit := t.splitNodeIfNeeded(parent)

	// Continue adjusting up the tree
	t.adjustTree(parent, parentSplit)

}

// NewEntry creates an entry Node with data.
func NewEntry(data GeoReferenced) *Node {
	bounds := data.Bounds()

	newEntry := &Node{
		Data:   data,
		MBR:    Rect{bounds[0], bounds[1], bounds[2], bounds[3]},
		IsLeaf: true,
	}

	return newEntry
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

	// Since the leaf now contains data nodes as children, it should be marked as an internal node
	leaf.IsLeaf = false

	// Split if the leaf overflows
	splitNode := t.splitNodeIfNeeded(leaf)

	// propagate changes upward
	t.adjustTree(leaf, splitNode)

}

// pickSeeds gives the two entries that are the farthest apart
func (t *RTree) pickSeeds(nodeA *Node) [2]*Node {

	seeds := [2]*Node{}

	var maxEnlargement float64

	// Pick the entries that (would waste the more area if put together).
	for i := 0; i < len(nodeA.Entries); i++ {
		for j := 0; j < len(nodeA.Entries); j++ {
			enlargement := nodeA.Entries[i].MBR.Enlargement(nodeA.Entries[j].MBR)
			if enlargement > maxEnlargement {
				maxEnlargement = enlargement
				seeds[0] = nodeA.Entries[i]
				seeds[1] = nodeA.Entries[j]
			}
		}
	}

	return seeds
}

func computeMBR(entries []*Node) Rect {

	var mbr Rect

	if len(entries) == 0 {
		return mbr
	}

	mbr = entries[0].MBR

	for i := 1; i < len(entries); i++ {
		mbr.Expand(entries[i].MBR)
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
		MBR:     seeds[0].MBR,
		Entries: []*Node{seeds[0]},
		IsLeaf:  node.IsLeaf,
		Parent:  node.Parent,
	}

	groupB := &Node{
		MBR:     seeds[1].MBR,
		Entries: []*Node{seeds[1]},
		IsLeaf:  node.IsLeaf,
		Parent:  node.Parent,
	}

	// Collect the remaining entries that aren't seeds to distribute
	remaining := slices.DeleteFunc(node.Entries, func(entry *Node) bool {
		return entry == seeds[0] || entry == seeds[1]
	})

	// Distribute remaining entries between groupA and groupB
	for len(remaining) > 0 {

		next := t.pickNext(remaining, groupA, groupB)
		entry := remaining[next]
		remaining = append(remaining[:next], remaining[next+1:]...)

		targetNode := t.chooseGroup(entry, groupA, groupB)

		targetNode.Entries = append(targetNode.Entries, entry)
		targetNode.MBR.Expand(entry.MBR)
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
func (t *RTree) chooseGroup(entry, groupA, groupB *Node) *Node {

	// Ensure minimum number of entries is met
	if len(groupA.Entries) < t.minEntries {
		return groupA
	}
	if len(groupB.Entries) < t.minEntries {
		return groupB
	}

	// Now choose the one which requires the lease enlargement
	// If it's a tie, chose the one with smallest area
	enlargeA := groupA.MBR.Enlargement(entry.MBR)
	enlargeB := groupB.MBR.Enlargement(entry.MBR)

	if enlargeA < enlargeB {
		return groupA
	}

	if enlargeB < enlargeA {
		return groupB
	}

	if groupA.MBR.Area() < groupB.MBR.Area() {
		return groupA
	}

	return groupB
}

// pickNext returns the index of the entry with the greatest preference to be inserted in a group.
func (t *RTree) pickNext(entries []*Node, groupA *Node, groupB *Node) int {

	next := 0
	maxDiff := -1.0

	for i, entry := range entries {
		d1 := groupA.MBR.Enlargement(entry.MBR)
		d2 := groupB.MBR.Enlargement(entry.MBR)
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

// Query finds all items intersecting the given Rect
func (t *RTree) Query(searchRect Rect) []GeoReferenced {
	if t.root == nil {
		return nil
	}

	var results []GeoReferenced
	t.search(t.root, searchRect, &results)
	return results
}

func (t *RTree) Entries() []Node {

	var entries []Node

	stack := []*Node{t.root}

	for len(stack) > 0 {

		entry := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		for _, child := range entry.Entries {
			if child.Data != nil {
				entries = append(entries, *child)
				continue
			} else {
				stack = append(stack, child)
			}
		}

	}

	return entries
}

// search performs the recursive search
func (t *RTree) search(node *Node, searchRect Rect, results *[]GeoReferenced) {
	if !node.MBR.Intersects(&searchRect) {
		return
	}

	for _, entry := range node.Entries {
		if entry.Data != nil {
			// This is a data node
			if entry.MBR.Intersects(&searchRect) {
				*results = append(*results, entry.Data)
			}
		} else {
			// This is an internal node
			t.search(entry, searchRect, results)
		}
	}
}
