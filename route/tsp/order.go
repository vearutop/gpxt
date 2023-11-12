package tsp

import (
	"fmt"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/route/tsp/base"
	"github.com/vearutop/gpxt/route/tsp/geneticAlgorithm"
	"log"
	"math"
)

func Order(points []gpx.GPXPoint) {
	// Init TourManager
	tm := base.TourManager{}
	tm.NewTourManager()

	// Generate Cities
	var cities []base.City
	var bitch [][2]int

	for _, p := range points {
		b1 := [2]int{int(math.Abs(10000000 * p.Longitude)), int(math.Abs(10000000 * p.Latitude))}
		found := false
		for _, b := range bitch {
			if b == b1 {
				found = true
				break
			}
		}

		if !found {
			bitch = append(bitch, b1)
			cities = append(cities, base.GenerateCity(b1[0], b1[1]))
		}
	}

	fmt.Printf("BIT: %#v\n", bitch)

	// Add cities to TourManager
	for _, v := range cities {
		tm.AddCity(v)
	}

	numberOfGenerations := 100
	populationSize := 600

	tspGA(&tm, numberOfGenerations, populationSize)
}

// tspGA : Travelling sales person with genetic algorithm
// input :- TourManager, Number of generations
func tspGA(tm *base.TourManager, gen int, popSize int) {
	p := base.Population{}
	p.InitPopulation(popSize, *tm)

	// Get initial fittest tour and it's tour distance
	fmt.Println("Start....")
	iFit := p.GetFittest()
	iTourDistance := iFit.TourDistance()
	//fmt.Println("Initial tour distance: ", iTourDistance)

	// Map to store fittest tours
	fittestTours := make([]base.Tour, 0, gen+1)
	fittestTours = append(fittestTours, *iFit)
	// Evolve population "gen" number of times
	for i := 1; i < gen+1; i++ {
		log.Println("Generation ", i)
		p = geneticAlgorithm.EvolvePopulation(p)
		ft := *p.GetFittest()
		fmt.Println("Distance:", ft.TourDistance())
		// Store fittest for each generation
		fittestTours = append(fittestTours, ft)
	}
	// Get final fittest tour and tour distance
	fmt.Println("Evolution completed")
	fFit := p.GetFittest()
	fTourDistance := fFit.TourDistance()

	//fmt.Println("Print and save image of fittest by generation-----------")
	//Remove old data
	//dname := fmt.Sprintf("%d", seed)
	//dname = filepath.Join(rootpath, dname)
	//os.RemoveAll(dname)

	// Store current best distance
	//lastBestTourDistance := iTourDistance
	// Plot Generation 0
	//visualization(iFit, 0, seed)
	//for gn, t := range fittestTours {
	//	if t.TourDistance() < lastBestTourDistance {
	//		lastBestTourDistance = t.TourDistance()
	//		fmt.Printf("Generation %v: %v\n", gn, lastBestTourDistance)
	//		Plot graph of points
	//visualization(&t, gn, seed)
	//}
	//}

	fmt.Println("Initial tour distance: ", iTourDistance)
	fmt.Println("Final tour distance: ", fTourDistance)

	log.Println("Evolution completed")
	log.Println("Initial tour distance: ", iTourDistance)
	log.Println("Final tour distance: ", fTourDistance)
}
