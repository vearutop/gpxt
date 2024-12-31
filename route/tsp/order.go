// Package tsp solves Traveling Salesman Problem with genetic algorithm.
package tsp

import (
	"github.com/vearutop/gpxt/gpx"
	"github.com/vearutop/gpxt/route/tsp/internal/base"
	"github.com/vearutop/gpxt/route/tsp/internal/ga"
)

// Defaults.
const (
	DefaultNumberOfGenerations = 100
	DefaultPopulationSize      = 600
)

// Order solves traveling sales person problem with genetic algorithm.
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

// tspGA solves traveling sales person problem with genetic algorithm.
// It returns ordered points, initial and final distances.
func tspGA(tm *base.TourManager, gen int, popSize int) ([]gpx.Point, float64, float64) {
	p := base.Population{}
	p.InitPopulation(popSize, *tm)

	// Get initial fittest tour and it's tour distance
	iFit := p.GetFittest()
	iTourDistance := iFit.TourDistance()
	// fmt.Println("Initial tour distance: ", iTourDistance)

	// Evolve population "gen" number of times
	for i := 1; i < gen+1; i++ {
		p = ga.EvolvePopulation(p)
	}

	// Get final fittest tour and tour distance
	fFit := p.GetFittest()
	fTourDistance := fFit.TourDistance()

	return fFit.Points(), iTourDistance, fTourDistance
}
