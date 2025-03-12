package rtree_test

import (
	"ralbero"
	"testing"
)

func TestNewRect(t *testing.T) {
	rect := rtree.NewRect(0, 0, 10, 10)
	if rect.MinX != 0 || rect.MinY != 0 || rect.MaxX != 10 || rect.MaxY != 10 {
		t.Errorf("Expected (0,0,10,10) but got (%f,%f,%f,%f)", rect.MinX, rect.MinY, rect.MaxX, rect.MaxY)
	}
}

func TestExpand(t *testing.T) {
	rect := rtree.NewRect(0, 0, 10, 10)
	otherRect := rtree.Rect{MinX: 5, MinY: 5, MaxX: 15, MaxY: 15}
	rect.Expand(otherRect)
	if rect.MinX != 0 || rect.MinY != 0 || rect.MaxX != 15 || rect.MaxY != 15 {
		t.Errorf("Expected (0,0,15,15) but got (%f,%f,%f,%f)", rect.MinX, rect.MinY, rect.MaxX, rect.MaxY)
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		rect          *rtree.Rect
		contained     *rtree.Rect
		shouldContain bool
	}{
		{
			rtree.NewRect(0, 0, 10, 10),
			rtree.NewRect(2, 2, 8, 8),
			true,
		},
		{
			rtree.NewRect(0, 0, 10, 10),
			rtree.NewRect(5, 5, 12, 12),
			false,
		},
		{
			rtree.NewRect(-10, -10, 10, 10),
			rtree.NewRect(5, 5, 10, 10),
			true,
		},
	}

	for _, tt := range tests {
		t.Run("Contains", func(t *testing.T) {
			result := tt.rect.Contains(*tt.contained)
			if result != tt.shouldContain {
				t.Errorf("Expected %v but got %v", tt.shouldContain, result)
			}
		})
	}
}

func TestIntersects(t *testing.T) {
	tests := []struct {
		rect            *rtree.Rect
		intersect       *rtree.Rect
		shouldIntersect bool
	}{
		{
			rtree.NewRect(0, 0, 10, 10),
			rtree.NewRect(5, 5, 15, 15),
			true,
		},
		{
			rtree.NewRect(0, 0, 10, 10),
			rtree.NewRect(15, 15, 20, 20),
			false,
		},
		{
			rtree.NewRect(-10, -10, 10, 10),
			rtree.NewRect(9, 9, 20, 20),
			true,
		},
	}

	for _, tt := range tests {
		t.Run("Intersects", func(t *testing.T) {
			result := tt.rect.Intersects(*tt.intersect)
			if result != tt.shouldIntersect {
				t.Errorf("Expected %v but got %v", tt.shouldIntersect, result)
			}
		})
	}
}

func TestArea(t *testing.T) {
	rect := rtree.NewRect(0, 0, 10, 10)
	expectedArea := 100.0
	if area := rect.Area(); area != expectedArea {
		t.Errorf("Area failed, expected %f but got %f", expectedArea, area)
	}
}

func TestEnlargement(t *testing.T) {
	rect := rtree.NewRect(0, 0, 10, 10)
	otherRect := rtree.Rect{MinX: 5, MinY: 5, MaxX: 15, MaxY: 15}
	expectedEnlargement := 125.0 // (15 * 15) - (10 * 10)
	if enlargement := rect.Enlargement(otherRect); enlargement != expectedEnlargement {
		t.Errorf("Enlargement failed, expected %f but got %f", expectedEnlargement, enlargement)
	}
}
