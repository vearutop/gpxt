// Package base holds basic entities.
package base

import "github.com/tkrajina/gpxgo/gpx"

// TourManager contains list of points to be visited.
type TourManager struct {
	points []gpx.Point
}

// NewTourManager creates TourManager.
func NewTourManager() *TourManager {
	return &TourManager{
		points: make([]gpx.Point, 0, 50),
	}
}

// AddPoint appends a point.
func (a *TourManager) AddPoint(c gpx.Point) {
	a.points = append(a.points, c)
}

func (a *TourManager) getPoint(i int) gpx.Point {
	return a.points[i]
}

// NumberOfPoints returns number of points.
func (a *TourManager) NumberOfPoints() int {
	return len(a.points)
}
