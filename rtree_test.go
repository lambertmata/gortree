package rtree_test

import (
	"log/slog"
	"ralbero"
	"strconv"
	"testing"
)

type Location struct {
	Name        string
	Coordinates [2]float64
}

func (l *Location) ID() string {
	return l.Name
}

func (l *Location) BoundingBox() rtree.Rect {
	x := l.Coordinates[0]
	y := l.Coordinates[1]

	rect := rtree.NewRect(x, y, x, y)

	return *rect
}

var cityLocations = []Location{
	{"Genova", [2]float64{8.928275776757602, 44.40716297481325}},
	{"Milan", [2]float64{9.19188426947727, 45.467509939027025}},
	{"Rome", [2]float64{12.49928631809945, 41.91961251548011}},
	{"Geneve", [2]float64{6.1517749934533015, 46.21514311923974}},
	{"Paris", [2]float64{2.3522, 48.8566}},
	{"London", [2]float64{-0.1276, 51.5074}},
	{"New York", [2]float64{-74.0060, 40.7128}},
	{"Tokyo", [2]float64{139.6917, 35.6895}},
	{"Berlin", [2]float64{13.4050, 52.5200}},
	{"Sydney", [2]float64{151.2093, -33.8688}},
	{"Dubai", [2]float64{55.2708, 25.276987}},
	{"Rio de Janeiro", [2]float64{-43.1729, -22.9068}},
	{"Los Angeles", [2]float64{-118.2437, 34.0522}},
	{"Shanghai", [2]float64{121.4737, 31.2304}},
	{"Hong Kong", [2]float64{114.1694, 22.3193}},
	{"Singapore", [2]float64{103.8198, 1.3521}},
	{"Bangkok", [2]float64{100.5167, 13.7563}},
	{"Mexico City", [2]float64{-99.1332, 19.4326}},
}

var WholeWorld = rtree.NewRect(
	-180,
	-90,
	180,
	90,
)

var NorthAmerica = rtree.NewRect(
	-168.0,
	5.0,
	-52.0,
	83.0,
)

func TestNewRTree(t *testing.T) {
	rt := rtree.NewRTree()

	if rt.Min() != rtree.MinEntries {
		t.Errorf("Expected rtree.MinEntries %d, got %d", rt.Min(), rtree.MinEntries)
	}

	if rt.Max() != rtree.MaxEntries {
		t.Errorf("Expected rtree.MaxEntries %d, got %d", rt.Max(), rtree.MaxEntries)
	}
}

func TestNewRTreeWithMinMax(t *testing.T) {

	rt, err := rtree.NewRTreeWithMinMax(1, 5)

	if err == nil {
		t.Errorf("Expected error for min entries = 1")
	}

	rt, err = rtree.NewRTreeWithMinMax(4, 1)

	if err == nil {
		t.Errorf("Expected error for max entries = 1")
	}

	minEntries := 2
	maxEntries := 8

	rt, err = rtree.NewRTreeWithMinMax(minEntries, maxEntries)

	if rt.Min() != minEntries {
		t.Errorf("Expected rtree.MinEntries %d, got %d", rt.Min(), minEntries)
	}

	if rt.Max() != maxEntries {
		t.Errorf("Expected rtree.MaxEntries %d, got %d", rt.Max(), maxEntries)
	}
}

func TestRTree_Insert(t *testing.T) {

	rt := rtree.NewRTree()

	for _, location := range cityLocations {
		rt.Insert(&location)
	}

	insertedEntries := rt.Entries()

	if len(insertedEntries) != len(cityLocations) {
		t.Errorf("Expected %d entries, got %d", len(cityLocations), len(insertedEntries))
	}
}

func TestRTree_Query(t *testing.T) {

	rt := rtree.NewRTree()

	testCases := []struct {
		Name     string
		Rect     rtree.Rect
		Expected int
	}{
		{"Whole World", *WholeWorld, 18},
		{"North America", *NorthAmerica, 3},
		{"Empty Rect", rtree.Rect{}, 0},
	}

	for _, location := range cityLocations {
		rt.Insert(&location)
	}

	for _, testCase := range testCases {
		foundEntries := rt.Query(testCase.Rect)
		if len(foundEntries) != testCase.Expected {
			for i, entry := range foundEntries {
				slog.Info("found", strconv.Itoa(i), entry.ID())
			}

			t.Errorf("Expected %d entries in %s, got %d", testCase.Expected, testCase.Name, len(foundEntries))
		}
	}

}
