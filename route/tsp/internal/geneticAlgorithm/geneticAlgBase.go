package geneticAlgorithm

import (
	"math/rand"

	"github.com/vearutop/gpxt/route/tsp/internal/base"
)

// Genetic Algorithm Parameters
var (
	mutationRate                = 0.015
	tournamentSize              = 10
	elitism                     = true
	randomCrossoverRate         = false
	defCrossoverRate    float32 = 0.7
)

func CrossoverRate() float32 {
	if randomCrossoverRate {
		return rand.Float32()
	}
	return defCrossoverRate
}

// Crossover  performs multipoint cross over with 2 parents with an
// assumption that parents have equal size.
func Crossover(p1 base.Tour, p2 base.Tour) base.Tour {
	// Size
	size := p1.TourSize()
	// Child Tour
	c := base.Tour{}
	c.InitTour(size)

	// Number of crossover
	nc := int(CrossoverRate() * float32(size))
	if nc == 0 {
		// log.Println("no crossover")
		return p1
	}
	// Start positions of cross over for parent 1
	sp := int(rand.Float32() * float32(size))
	// End position of cross over for parent 1
	ep := (sp + nc) % size
	// Parent 2 slots
	p2s := make([]int, 0, size-nc)
	// log.Println(size, sp, nc, ep) // For debugging
	// Populate child with parent 1
	if sp < ep {
		for i := 0; i < size; i++ {
			if i >= sp && i < ep {
				c.SetPoint(i, p1.GetPoint(i))
			} else {
				p2s = append(p2s, i)
			}
		}
	} else if sp > ep {
		for i := 0; i < size; i++ {
			if !(i >= ep && i < sp) {
				c.SetPoint(i, p1.GetPoint(i))
			} else {
				p2s = append(p2s, i)
			}
		}
	}

	j := 0
	// Populate child with parent 2 cities that are missing
	for i := 0; i < size; i++ {
		// Check if child contains city
		if !c.ContainsPoint(p2.GetPoint(i)) {
			c.SetPoint(p2s[j], p2.GetPoint(i))
			j++
		}
	}

	return c
}

// Mutation performs swap mutation.
// Chance of mutation for each City based on mutation rate.
func Mutation(in *base.Tour) {
	// for each city
	for p1 := 0; p1 < in.TourSize(); p1++ {
		if rand.Float64() < mutationRate {
			// Select 2nd city to perform swap
			p2 := int(float64(in.TourSize()) * rand.Float64())
			// log.Println("Mutation occured", p1, "swap", p2)
			// Temp store city
			c1 := in.GetPoint(p1)
			c2 := in.GetPoint(p2)
			// Swap Cities
			in.SetPoint(p1, c2)
			in.SetPoint(p2, c1)
		}
	}
}

// TournamentSelection : select a group at random and pick the best parent
func TournamentSelection(pop base.Population) base.Tour {
	tourny := base.Population{}
	tourny.InitEmpty(tournamentSize)

	for i := 0; i < tournamentSize; i++ {
		r := int(rand.Float64() * float64(pop.PopulationSize()))
		tourny.SaveTour(i, *pop.GetTour(r))
	}
	// fittest tour
	fTour := tourny.GetFittest()
	return *fTour
}

// EvolvePopulation evolves population by:
//   - Selecting 2 parents using tournament selection
//   - Perform crossover to obtain child
//   - Mutate child based on probability
//   - return new population
func EvolvePopulation(pop base.Population) base.Population {
	npop := base.Population{}
	npop.InitEmpty(pop.PopulationSize())

	popOffset := 0
	if elitism {
		npop.SaveTour(0, *pop.GetFittest())
		popOffset = 1
	}

	for i := popOffset; i < npop.PopulationSize(); i++ {
		p1 := TournamentSelection(pop)
		p2 := TournamentSelection(pop)
		child := Crossover(p1, p2)
		Mutation(&child)
		npop.SaveTour(i, child)
	}
	return npop
}
