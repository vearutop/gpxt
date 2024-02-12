package tsp

import (
	"fmt"

	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/route/tsp/base"
	"github.com/vearutop/gpxt/route/tsp/geneticAlgorithm"
)

func Order(points []gpx.GPXPoint) []gpx.Point {
	// Init TourManager
	tm := base.TourManager{}
	tm.NewTourManager()

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

	// Add cities to TourManager
	for _, v := range cities {
		tm.AddCity(v)
	}

	numberOfGenerations := 100
	populationSize := 600

	return tspGA(&tm, numberOfGenerations, populationSize)
}

// tspGA : Travelling sales person with genetic algorithm
// input :- TourManager, Number of generations
func tspGA(tm *base.TourManager, gen int, popSize int) []gpx.Point {
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

	fmt.Println("Initial tour distance: ", iTourDistance)
	fmt.Println("Final tour distance: ", fTourDistance)

	return fFit.Points()
}
