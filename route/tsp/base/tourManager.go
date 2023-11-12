package base

import "github.com/tkrajina/gpxgo/gpx"

// ToueManager : Contains list of of cities to be visited
type TourManager struct {
	destCities []gpx.Point
}

// NewTourManager : Initialize TourManager
func (a *TourManager) NewTourManager() {
	a.destCities = make([]gpx.Point, 0, 50)
}

func (a *TourManager) AddCity(c gpx.Point) {
	a.destCities = append(a.destCities, c)
}

func (a *TourManager) GetCity(i int) gpx.Point {
	return a.destCities[i]
}

func (a *TourManager) NumberOfCities() int {
	return len(a.destCities)
}
