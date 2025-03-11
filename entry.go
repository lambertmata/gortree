package rtree

type Entry struct {
	Data        GeoReferenced
	BoundingBox Rect
	Parent      *Node
}

// NewEntry creates an entry Node with data.
func NewEntry(data GeoReferenced) *Entry {

	newEntry := &Entry{
		Data:        data,
		BoundingBox: data.BoundingBox(),
	}

	return newEntry
}

func (e *Entry) String() string {
	return e.Data.ID()
}
