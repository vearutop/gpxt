package tsp

import (
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/route/tsp/base"
	"github.com/vearutop/gpxt/route/tsp/geneticAlgorithm"
)

const (
	DefaultNumberOfGenerations = 100
	DefaultPopulationSize      = 600
)

// Order solves travelling sales person problem with genetic algorithm.
// It returns ordered points, initial and final distances.
// Use DefaultNumberOfGenerations and DefaultPopulationSize when in doubt.
func Order(points []gpx.GPXPoint, numberOfGenerations, populationSize int) ([]gpx.Point, float64, float64) {
	// Init TourManager
	tm := base.NewTourManager()

	// Prepare points
	var cities []gpx.Point

	for _, p := range points {
		p1 := p.Point
		found := false
		// Deduplicate points.
		for _, b := range cities {
			if b == p1 {
				found = true
				break
			}
		}

		if !found {
			cities = append(cities, p.Point)
		}
	}

	// Add points to TourManager
	for _, v := range cities {
		tm.AddPoint(v)
	}

	return tspGA(tm, numberOfGenerations, populationSize)
}

// tspGA solves travelling sales person problem with genetic algorithm.
// It returns ordered points, initial and final distances.
func tspGA(tm *base.TourManager, gen int, popSize int) ([]gpx.Point, float64, float64) {
	p := base.Population{}
	p.InitPopulation(popSize, *tm)

	// Get initial fittest tour and it's tour distance
	iFit := p.GetFittest()
	iTourDistance := iFit.TourDistance()
	// fmt.Println("Initial tour distance: ", iTourDistance)

	// Map to store fittest tours
	fittestTours := make([]base.Tour, 0, gen+1)
	fittestTours = append(fittestTours, *iFit)

	// Evolve population "gen" number of times
	for i := 1; i < gen+1; i++ {
		p = geneticAlgorithm.EvolvePopulation(p)
		ft := *p.GetFittest()
		// Store fittest for each generation
		fittestTours = append(fittestTours, ft)
	}

	// Get final fittest tour and tour distance
	fFit := p.GetFittest()
	fTourDistance := fFit.TourDistance()

	return fFit.Points(), iTourDistance, fTourDistance
}
