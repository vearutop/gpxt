package base

import "github.com/tkrajina/gpxgo/gpx"

// TourManager contains list of points to be visited.
type TourManager struct {
	points []gpx.Point
}

// NewTourManager : Initialize TourManager
func NewTourManager() *TourManager {
	return &TourManager{
		points: make([]gpx.Point, 0, 50),
	}
}

func (a *TourManager) AddPoint(c gpx.Point) {
	a.points = append(a.points, c)
}

func (a *TourManager) GetPoint(i int) gpx.Point {
	return a.points[i]
}

func (a *TourManager) NumberOfPoints() int {
	return len(a.points)
}
