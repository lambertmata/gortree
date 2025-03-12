package gortree

import (
	"errors"
	"fmt"
	"math"
	"slices"
)

type Spatial interface {
	BoundingBox() Rect
	ID() string
}

type RTree struct {
	root       *node
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
		root: &node{
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
func (t *RTree) chooseLeaf(n *node, boundingBox Rect) *node {

	// The tree is descended until a leaf is reached, by selecting the child node which requires the least enlargement
	// to contain rect.
	// If a tie occurs, meaning two child have the same enlargement, the node with the smallest area is selected.
	if n.IsLeaf {
		return n
	}

	var bestNode *node
	minEnlargement := math.MaxFloat64

	for _, child := range n.Children {

		if n.IsLeaf {
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
		return n
	}

	return t.chooseLeaf(bestNode, boundingBox)
}

// updateNodeMBR Using current entries MBRs it updated the node BoundingBox.
func (t *RTree) updateNodeMBR(n *node) {
	n.BoundingBox = computeNodesMBR(n.Children)
}

// updateMBRsUpward updates MBRs starting from node up to the root.
func (t *RTree) updateMBRsUpward(n *node) {
	for n != nil {
		t.updateNodeMBR(n)
		n = n.Parent
	}
}

// adjustTree updates the MBRs up the tree after an insertion
func (t *RTree) adjustTree(n *node, splitNode *node) {

	// Case 1: If no split occurred, just update MBRs up the tree
	if splitNode == nil {
		t.updateMBRsUpward(n)
		return
	}

	// Case 2: Root split
	if n.Parent == nil {

		// Create a new root
		newRoot := &node{
			IsLeaf:   false,
			Children: []*node{n, splitNode},
		}

		// Update parent references
		n.Parent = newRoot
		splitNode.Parent = newRoot

		// Update tree's root
		t.root = newRoot

		// Update the BoundingBox of the new root
		newRoot.BoundingBox = computeNodesMBR(newRoot.Children)

		return
	}

	// Case 3: Split occurred at non-root level

	// We need to add the new node to the parent and continue adjusting upward
	parent := n.Parent

	// Update the BoundingBox of the original node
	n.BoundingBox = computeNodesMBR(n.Children)

	// Add splitNode to parent
	parent.Children = append(parent.Children, splitNode)
	splitNode.Parent = parent

	// Check if parent needs splitting
	parentSplit := t.splitNodeIfNeeded(parent)

	// Continue adjusting up the tree
	t.adjustTree(parent, parentSplit)

}

// Insert adds a new item to the tree.
func (t *RTree) Insert(data Spatial) {

	// Create the new entry node
	e := newLeafNode(data)

	// Find the best leaf node to insert the new entry node.
	leaf := t.chooseLeaf(t.root, e.BoundingBox)

	// Add entry node to leaf
	leaf.Children = append(leaf.Children, e)
	e.Parent = leaf

	// Split if the leaf overflows
	splitNode := t.splitNodeIfNeeded(leaf)

	// propagate changes upward
	t.adjustTree(leaf, splitNode)

}

// pickSeeds selects the two entries that are the farthest apart.
func (t *RTree) pickSeeds(n *node) [2]*node {

	seeds := [2]*node{}
	maxEnlargement := 0.0

	// Pick the entries that would waste the more area if put together
	for i := 0; i < len(n.Children); i++ {
		for j := 0; j < len(n.Children); j++ {
			enlargement := n.Children[i].BoundingBox.Enlargement(n.Children[j].BoundingBox)
			if enlargement > maxEnlargement {
				maxEnlargement = enlargement
				seeds[0] = n.Children[i]
				seeds[1] = n.Children[j]
			}
		}
	}

	return seeds
}

// computeNodesMBR returns the minimum bounding rectangle containing all nodes.
func computeNodesMBR(nodes []*node) Rect {
	var mbr Rect
	for _, n := range nodes {
		mbr.Expand(n.BoundingBox)
	}
	return mbr
}

// splitNodeIfNeeded performs splitNode only when node is overflowing.
func (t *RTree) splitNodeIfNeeded(node *node) *node {
	if !t.nodeOverflowing(node) {
		return nil
	}
	return t.splitNode(node)
}

// splitNode performs quadratic split of the given node.
func (t *RTree) splitNode(n *node) *node {

	// Pick two entries that are furthest apart
	seeds := t.pickSeeds(n)

	// Create two groups with a seed each
	a := &node{
		BoundingBox: seeds[0].BoundingBox,
		Children:    []*node{seeds[0]},
		IsLeaf:      n.IsLeaf,
		Parent:      n.Parent,
	}

	b := &node{
		BoundingBox: seeds[1].BoundingBox,
		Children:    []*node{seeds[1]},
		IsLeaf:      n.IsLeaf,
		Parent:      n.Parent,
	}

	// Collect remaining entries to distribute (that aren't seeds)
	remaining := slices.DeleteFunc(n.Children, func(e *node) bool {
		return e == seeds[0] || e == seeds[1]
	})

	// Distribute remaining entries between a and b
	for len(remaining) > 0 {

		idx := t.pickNext(remaining, a, b)
		e := remaining[idx]
		remaining = append(remaining[:idx], remaining[idx+1:]...)

		target := t.chooseGroup(e, a, b)

		target.Children = append(target.Children, e)
		target.BoundingBox.Expand(e.BoundingBox)
	}

	// Replace original node with group a
	*n = *a

	// Update parent pointers
	t.adjustEntriesParent(n)
	t.adjustEntriesParent(b)

	// Return b as the new split node
	return b
}

// chooseGroup returns the group where entry should be assigned to.
func (t *RTree) chooseGroup(e *node, a, b *node) *node {

	// Ensure minimum number of entries is met
	if len(a.Children) < t.minEntries {
		return a
	}
	if len(b.Children) < t.minEntries {
		return b
	}

	// Now choose the one which requires the lease enlargement
	// If it's a tie, chose the one with smallest area
	enlargeA := a.BoundingBox.Enlargement(e.BoundingBox)
	enlargeB := b.BoundingBox.Enlargement(e.BoundingBox)

	if enlargeA < enlargeB {
		return a
	}

	if enlargeB < enlargeA {
		return b
	}

	if a.BoundingBox.Area() < b.BoundingBox.Area() {
		return a
	}

	return b
}

// pickNext returns the index of the entry with the greatest preference to be inserted in a group.
func (t *RTree) pickNext(entries []*node, groupA *node, groupB *node) int {

	next := 0
	maxDiff := -1.0

	for i, entry := range entries {
		area1 := groupA.BoundingBox.Enlargement(entry.BoundingBox)
		area2 := groupB.BoundingBox.Enlargement(entry.BoundingBox)
		diff := math.Abs(area1 - area2)
		if diff > maxDiff {
			maxDiff = diff
			next = i
		}
	}

	return next
}

// adjustEntriesParent updates the node entries such that their Parent pointer points to the node.
func (t *RTree) adjustEntriesParent(node *node) {
	for _, c := range node.Children {
		c.Parent = node
	}
}

// removeNodeFromParent removes node from parent, without knowing its index in the parent.Children slice.
func (t *RTree) removeNodeFromParent(parent, node *node) error {

	if parent == nil {
		return errors.New("failed to remove node without parent")
	}

	for i, c := range parent.Children {
		if c != node {
			continue
		}
		// Remove current from parent
		parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
		break
	}

	return nil
}

// nodeOverflowing returns whether the current node has too many entries.
func (t *RTree) nodeOverflowing(n *node) bool {
	return len(n.Children) > t.maxEntries
}

// nodeUnderflowing returns whether the current node has too few entries.
func (t *RTree) nodeUnderflowing(n *node) bool {
	return len(n.Children) < t.minEntries
}

// collectLeafNodes returns the current node descendant leaf nodes.
func (t *RTree) collectLeafNodes(n *node) []*node {
	if n.IsLeaf {
		return n.Children
	}

	var leaves []*node
	for _, c := range n.Children {
		leaves = append(leaves, t.collectLeafNodes(c)...)
	}
	return leaves
}

// CondenseTree handles nodes with too few entries after deletion. It removes underflowing nodes and returns their
// entries so they can be reinserted.
func (t *RTree) condenseTree(n *node) []*node {

	var orphans []*node // Stores the node that will need to be reinserted
	cur := n            // The node where the delete took place

	// Repeat the process from current node all the way up to the root
	for cur.Parent != nil {

		parent := cur.Parent

		// Check if the current node has too few entries
		if t.nodeUnderflowing(cur) {

			_ = t.removeNodeFromParent(parent, cur)

			// Collect all leaf nodes that need to be reinserted
			if cur.IsLeaf {
				// Collect all entries
				orphans = append(orphans, cur.Children...)
			} else {
				// Collect all entries descending the subtree
				orphans = append(orphans, t.collectLeafNodes(cur)...)
			}

		} else {
			// Just update the current node bounding box
			t.updateNodeMBR(cur)
		}

		cur = parent
	}

	// Finally adjust the root bounding box as well
	t.updateNodeMBR(t.root)

	return orphans

}

// findLeaf starting from the root it searches the given data by ID, narrowing down the results using the bounding box.
func (t *RTree) findLeaf(data Spatial) *node {

	if t.root == nil {
		return nil
	}

	stack := []*node{t.root}

	bbox := data.BoundingBox()
	id := data.ID()

	// Traverse the tree starting from the root
	for len(stack) > 0 {

		n := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		// Follow internal nodes paths only when the target bounding box is guaranteed to be in the subtree
		if !n.IsLeaf {
			if n.BoundingBox.Intersects(bbox) {
				stack = append(stack, n.Children...)
			}
			continue
		}

		// Leaf node here. Just return the entry node if the ID matches
		for _, e := range n.Children {
			if e.Data != nil && e.Data.ID() == id {
				return n
			}
		}

	}

	return nil
}

// Delete deletes the entry from the tree by the data ID.
func (t *RTree) Delete(data Spatial) error {

	// Find the leaf node which contains data ID
	leaf := t.findLeaf(data)

	if leaf == nil {
		return errors.New("node to delete not found")
	}

	// Remove the entry from the leaf node
	leaf.Children = slices.DeleteFunc(leaf.Children, func(entry *node) bool {
		return entry.Data != nil && entry.Data.ID() == data.ID()
	})

	// Handle the underflow after deletion and collect orphaned entries
	orphans := t.condenseTree(leaf)

	// Reinsert the orphaned entries
	for _, o := range orphans {
		t.Insert(o.Data)
	}

	// Make the leaf the new root if it's the only one child
	if leaf.Parent == nil && len(leaf.Children) == 1 {
		t.root = leaf.Children[0]
		t.root.Parent = nil
	}

	return nil
}
