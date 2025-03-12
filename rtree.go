package rtree

import (
	"errors"
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
func (t *RTree) chooseLeaf(node *Node, boundingBox Rect) *Node {

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

		enlargement := child.BoundingBox.Enlargement(boundingBox)

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

	return t.chooseLeaf(bestNode, boundingBox)
}

// updateNodeMBR Using current entries MBRs it updated the node BoundingBox.
func (t *RTree) updateNodeMBR(node *Node) {
	node.BoundingBox = computeNodesMBR(node.Children)
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
	newEntry := NewLeafNode(data)

	// Find the best leaf node to insert the new entry node.
	leaf := t.chooseLeaf(t.root, newEntry.BoundingBox)

	// Add entry node to leaf
	leaf.Children = append(leaf.Children, newEntry)
	newEntry.Parent = leaf

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
	for i := 0; i < len(nodeA.Children); i++ {
		for j := 0; j < len(nodeA.Children); j++ {
			enlargement := nodeA.Children[i].BoundingBox.Enlargement(nodeA.Children[j].BoundingBox)
			if enlargement > maxEnlargement {
				maxEnlargement = enlargement
				seeds[0] = nodeA.Children[i]
				seeds[1] = nodeA.Children[j]
			}
		}
	}

	return seeds
}

// computeNodesMBR returns the bounding box to contain all the nodes.
func computeNodesMBR(nodes []*Node) Rect {
	var mbr Rect
	for i := 0; i < len(nodes); i++ {
		mbr.Expand(nodes[i].BoundingBox)
	}
	return mbr
}

// splitNodeIfNeeded preforms splitNode only when node is overflowing.
func (t *RTree) splitNodeIfNeeded(node *Node) *Node {
	if !t.nodeOverflowing(node) {
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
		Children:    []*Node{seeds[0]},
		IsLeaf:      node.IsLeaf,
		Parent:      node.Parent,
	}

	groupB := &Node{
		BoundingBox: seeds[1].BoundingBox,
		Children:    []*Node{seeds[1]},
		IsLeaf:      node.IsLeaf,
		Parent:      node.Parent,
	}

	// Collect the remaining entries that aren't seeds to distribute
	remaining := slices.DeleteFunc(node.Children, func(entry *Node) bool {
		return entry == seeds[0] || entry == seeds[1]
	})

	// Distribute remaining entries between groupA and groupB
	for len(remaining) > 0 {

		next := t.pickNext(remaining, groupA, groupB)
		entry := remaining[next]
		remaining = append(remaining[:next], remaining[next+1:]...)

		targetNode := t.chooseGroup(entry, groupA, groupB)

		targetNode.Children = append(targetNode.Children, entry)
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
func (t *RTree) chooseGroup(entry *Node, groupA, groupB *Node) *Node {

	// Ensure minimum number of entries is met
	if len(groupA.Children) < t.minEntries {
		return groupA
	}
	if len(groupB.Children) < t.minEntries {
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
func (t *RTree) pickNext(entries []*Node, groupA *Node, groupB *Node) int {

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
	for _, child := range node.Children {
		child.Parent = node
	}
}

// removeNodeFromParent removes node from parent, without knowing its index in the parent.Children slice.
func (t *RTree) removeNodeFromParent(parent, node *Node) error {

	if parent == nil {
		return errors.New("failed to remove node without parent")
	}

	for i, child := range parent.Children {
		if child != node {
			continue
		}
		// Remove current from parent
		parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
		break
	}

	return nil
}

// nodeOverflowing returns whether the current node has too many entries.
func (t *RTree) nodeOverflowing(node *Node) bool {
	return len(node.Children) > t.maxEntries
}

// nodeUnderflowing returns whether the current node has too few entries.
func (t *RTree) nodeUnderflowing(node *Node) bool {
	return len(node.Children) < t.minEntries
}

// collectLeafNodes returns the current node descendant leaf nodes.
func (t *RTree) collectLeafNodes(node *Node) []*Node {
	if node.IsLeaf {
		return node.Children
	}

	var leafNodes []*Node
	for _, child := range node.Children {
		leafNodes = append(leafNodes, t.collectLeafNodes(child)...)
	}
	return leafNodes
}

// CondenseTree handles nodes with too few entries after deletion. It removes underflowing nodes and returns their
// entries so they can be reinserted.
func (t *RTree) condenseTree(node *Node) []*Node {

	var orphanedEntries []*Node // Stores the node that will need to be reinserted
	currentNode := node         // The node where the delete took place

	// Repeat the process from current node all the way up to the root
	for currentNode.Parent != nil {

		parent := currentNode.Parent

		// Check if the current node has too few entries
		if t.nodeUnderflowing(currentNode) {

			_ = t.removeNodeFromParent(parent, currentNode)

			// Collect all leaf nodes that need to be reinserted
			if currentNode.IsLeaf {
				// Collect all entries
				orphanedEntries = append(orphanedEntries, currentNode.Children...)
			} else {
				// Collect all entries descending the subtree
				orphanedEntries = append(orphanedEntries, t.collectLeafNodes(currentNode)...)
			}

		} else {
			// Just update the current node bounding box
			t.updateNodeMBR(currentNode)
		}

		currentNode = parent
	}

	// Finally adjust the root bounding box as well
	t.updateNodeMBR(t.root)

	return orphanedEntries

}

// findLeaf starting from the root it searches the given data by ID, narrowing down the results using the bounding box.
func (t *RTree) findLeaf(data GeoReferenced) *Node {

	if t.root == nil {
		return nil
	}

	stack := []*Node{t.root}

	targetBoundingBox := data.BoundingBox()
	targetID := data.ID()

	// Traverse the tree starting from the root
	for len(stack) > 0 {

		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Follow internal nodes paths only when the target bounding box is guaranteed to be in the subtree
		if !node.IsLeaf {
			if node.BoundingBox.Intersects(targetBoundingBox) {
				stack = append(stack, node.Children...)
			}
			continue
		}

		// Leaf node here. Just return the entry node if the ID matches
		for _, leaf := range node.Children {
			if leaf.Data != nil && leaf.Data.ID() == targetID {
				return node
			}
		}

	}

	return nil
}

// Delete deletes the entry from the tree by the data ID.
func (t *RTree) Delete(data GeoReferenced) error {

	// Find the leaf node which contains data ID
	leaf := t.findLeaf(data)

	if leaf == nil {
		return errors.New("node to delete not found")
	}

	// Remove the entry from the leaf node
	leaf.Children = slices.DeleteFunc(leaf.Children, func(entry *Node) bool {
		return entry.Data != nil && entry.Data.ID() == data.ID()
	})

	// Handle the underflow after deletion.
	// If the node has too few entries, it will be removed and its entries returned to be inserted.
	// This is done recursively.
	orphanedEntries := t.condenseTree(leaf)

	// Reinsert the entries
	for _, orphan := range orphanedEntries {
		t.Insert(orphan.Data)
	}

	// If leaf is the only child of the root, compact the tree by making the leaf the root.
	if leaf.Parent == nil && len(leaf.Children) == 1 {
		t.root = leaf.Children[0]
		t.root.Parent = nil
	}

	return nil
}
