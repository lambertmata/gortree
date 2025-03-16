# gortree

An R-tree spatial index implementation in Go, for fast geospatial queries.

## Installation

```
go get github.com/lambertmata/gortree
```

## Usage

### Creating an R-tree

```go
// Create with default parameters
rt := gortree.NewRTree()

// Or with custom min/max entries
rt, err := gortree.NewRTreeWithMinMax(4, 20)
```

### Implementing the Spatial interface

All objects stored in the R-tree must implement the Spatial interface:

```go
type Spatial interface {
    BoundingBox() Rect
    ID() string
}
```

Example implementation:

```go

type Location struct {
    Name string
    Lat  float64
    Lon  float64
}

func (l *Location) ID() string {
    return l.Name
}

func (l *Location) BoundingBox() gortree.Rect {
    bounds := gortree.NewRect(l.Lon, l.Lat, l.Lon, l.Lat)
    return *bounds
}

```

### Basic Operations

```go
rt := gortree.NewRTree()

// Create spatial value
location := Location{
    Name: "Null Island",
    Lat:  0,
    Lon:  0,
}


// Insert an object
rt.Insert(&location)

// Search for the inserted location
queried := rt.Query(gortree.Rect{})

// Delete the inserted location
err := rt.Delete(location)

// Get all locations
all := rt.Entries()
```

## Features

- Spatial data structure for area-based and point queries
- Supports insert, delete, and search operations
- Based on the original R-tree algorithm (Guttman, 1984)

