package base

// Population is a collection of tours.
type Population struct {
	tours []Tour
}

// InitEmpty prepares population.
func (a *Population) InitEmpty(pSize int) {
	a.tours = make([]Tour, pSize)
}

// InitPopulation prepares population.
func (a *Population) InitPopulation(pSize int, tm TourManager) {
	a.tours = make([]Tour, pSize)

	for i := 0; i < pSize; i++ {
		nT := Tour{}
		nT.InitTourPoints(tm)
		a.SaveTour(i, nT)
	}
}

// SaveTour saves tour.
func (a *Population) SaveTour(i int, t Tour) {
	a.tours[i] = t
}

// GetTour returns specific tour.
func (a *Population) GetTour(i int) *Tour {
	return &a.tours[i]
}

// PopulationSize returns population size.
func (a *Population) PopulationSize() int {
	return len(a.tours)
}

// GetFittest returns the best known tour.
func (a *Population) GetFittest() *Tour {
	fittest := a.tours[0]
	// Loop through all tours taken by population and determine the fittest
	for i := 0; i < a.PopulationSize(); i++ {
		if fittest.normalizedFitness() <= a.GetTour(i).normalizedFitness() {
			fittest = *a.GetTour(i)
		}
	}

	return &fittest
}
