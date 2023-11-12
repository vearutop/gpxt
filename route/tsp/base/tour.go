package base

import (
	"fmt"
	"github.com/tkrajina/gpxgo/gpx"
	"strconv"
)

type Tour struct {
	tourPoints []gpx.Point
	fitness    float64
	distance   float64
}

// InitTour : Initialize tour with cities arranged randomly
func (a *Tour) InitTour(numberOfCities int) {
	a.tourPoints = make([]gpx.Point, numberOfCities)
}

// InitTourCities
func (a *Tour) InitTourCities(tm TourManager) {
	a.InitTour(tm.NumberOfCities())
	// Add all destination cities from TourManager to Tour
	for i := 0; i < tm.NumberOfCities(); i++ {
		a.SetCity(i, tm.GetCity(i))
	}
	// Shuffle cities in tour
	a.tourPoints = ShuffleCities(a.tourPoints)
}

// GetCity : Get city based on position in slice
func (a *Tour) GetCity(tourPosition int) gpx.Point {
	return a.tourPoints[tourPosition]
}

// SetCity : Set position of city in tour slice
func (a *Tour) SetCity(tourPosition int, c gpx.Point) {
	a.tourPoints[tourPosition] = c
	// Reset fitness if tour have been altered
	a.fitness = 0
	a.distance = 0
}

func (a *Tour) ResetFitnessDistance() {
	a.fitness = 0
	a.distance = 0
}

func (a *Tour) TourSize() int {
	return len(a.tourPoints)
}

// TourDistance : Calculates total distance traveled for this tour
func (a *Tour) TourDistance() float64 {
	if a.distance == 0 {
		td := float64(0)
		for i := 0; i < a.TourSize(); i++ {
			fromC := a.GetCity(i)
			var destC gpx.Point
			if i+1 < a.TourSize() {
				destC = a.GetCity(i + 1)
			} else {
				destC = a.GetCity(0)
			}
			td += fromC.Distance2D(&destC)
		}
		a.distance = td
	}
	return a.distance
}

func (a *Tour) Fitness() float64 {
	if a.fitness == 0 {
		a.fitness = 1 / a.TourDistance()
	}
	return a.fitness
}

func (a *Tour) ContainCity(c gpx.Point) bool {
	for _, cs := range a.tourPoints {
		if cs == c {
			return true
		}
	}
	return false
}

func (a Tour) String() string {
	s := "|"
	for i, c := range a.tourPoints {
		s += strconv.Itoa(i) + fmt.Sprintf("LON %.f, LAT %.f", c.Longitude, c.Latitude) + "|"
	}
	return s
}
