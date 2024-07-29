package base

import (
	"fmt"
	"math/rand"
	"strconv"

	"github.com/tkrajina/gpxgo/gpx"
)

// Tour is an arranged list of points.
type Tour struct {
	tourPoints []gpx.Point
	fitness    float64
	distance   float64
}

// InitTour initializes tour with points arranged randomly.
func (a *Tour) InitTour(numberOfPoints int) {
	a.tourPoints = make([]gpx.Point, numberOfPoints)
}

// Points returns list of points.
func (a *Tour) Points() []gpx.Point {
	return a.tourPoints
}

// InitTourPoints inits tour.
func (a *Tour) InitTourPoints(tm TourManager) {
	a.InitTour(tm.NumberOfPoints())

	// Add all destination points from TourManager to Tour
	for i := 0; i < tm.NumberOfPoints(); i++ {
		a.SetPoint(i, tm.getPoint(i))
	}

	// Shuffle points in tour
	a.tourPoints = shufflePoints(a.tourPoints)
}

// shufflePoints returns a shuffled []gpx.Point given input []gpx.Point.
func shufflePoints(in []gpx.Point) []gpx.Point {
	out := make([]gpx.Point, len(in), cap(in))
	perm := rand.Perm(len(in))

	for i, v := range perm {
		out[v] = in[i]
	}

	return out
}

// GetPoint gets point based on position in slice.
func (a *Tour) GetPoint(tourPosition int) gpx.Point {
	return a.tourPoints[tourPosition]
}

// SetPoint sets position of a point in tour slice.
func (a *Tour) SetPoint(tourPosition int, c gpx.Point) {
	a.tourPoints[tourPosition] = c
	// Reset fitness if tour have been altered
	a.fitness = 0
	a.distance = 0
}

// ResetFitnessDistance zeroes fitness and distance.
func (a *Tour) ResetFitnessDistance() {
	a.fitness = 0
	a.distance = 0
}

// TourSize return number of points in tour.
func (a *Tour) TourSize() int {
	return len(a.tourPoints)
}

// TourDistance calculates total distance traveled for this tour.
func (a *Tour) TourDistance() float64 {
	if a.distance == 0 {
		td := float64(0)

		for i := 0; i < a.TourSize(); i++ {
			fromC := a.GetPoint(i)
			destC := gpx.Point{}

			if i+1 < a.TourSize() {
				destC = a.GetPoint(i + 1)
			} else {
				destC = a.GetPoint(0)
			}

			td += fromC.Distance2D(&destC)
		}

		a.distance = td
	}

	return a.distance
}

func (a *Tour) normalizedFitness() float64 {
	if a.fitness == 0 {
		a.fitness = 1 / a.TourDistance()
	}

	return a.fitness
}

// ContainsPoint return true if point is in tour points.
func (a *Tour) ContainsPoint(c gpx.Point) bool {
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
