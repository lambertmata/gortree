package rtree

import "math"

type Rect struct {
	MinX, MinY, MaxX, MaxY float64
}

func NewRect(minX, minY, maxX, maxY float64) *Rect {
	return &Rect{
		minX, minY,
		maxX, maxY,
	}
}

// Expand Expands the current rect to contain otherRect
func (r *Rect) Expand(otherRect Rect) {
	r.MinX = math.Min(r.MinX, otherRect.MinX)
	r.MinY = math.Min(r.MinY, otherRect.MinY)
	r.MaxX = math.Max(r.MaxX, otherRect.MaxX)
	r.MaxY = math.Max(r.MaxY, otherRect.MaxY)
}

// Contains Checks if a rectangle contains another
func (r *Rect) Contains(other *Rect) bool {
	return r.MinX <= other.MinX && r.MaxX >= other.MaxX &&
		r.MinY <= other.MinY && r.MaxY >= other.MaxY
}

// Intersects Checks if a rectangle intersects another
func (r *Rect) Intersects(other *Rect) bool {
	return r.MinX <= other.MaxX && r.MaxX >= other.MinX &&
		r.MinY <= other.MaxY && r.MaxY >= other.MinY
}

// Area Returns the area of the rectangle
func (r *Rect) Area() float64 {
	width := r.MaxX - r.MinX
	height := r.MaxY - r.MinY
	return width * height
}

// Enlargement Returns the area enlargement required to container otherRect
func (r *Rect) Enlargement(otherRect Rect) float64 {
	area := r.Area()
	expandedRect := *r
	expandedRect.Expand(otherRect)
	expandedArea := expandedRect.Area()
	return expandedArea - area
}
