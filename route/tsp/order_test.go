package tsp_test

import (
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/route/tsp"
)

func TestOrder(t *testing.T) {
	gpxFile, err := gpx.ParseFile("testdata/photos-2023-10-faro.gpx")
	require.NoError(t, err)

	// Evolution completed
	// Initial tour distance:  4.92231483363616e+06
	// Final tour distance:  2.796963430731682e+06

	rand.Seed(1)
	_, initial, final := tsp.Order(gpxFile.Waypoints, tsp.DefaultNumberOfGenerations, tsp.DefaultPopulationSize)

	assert.InEpsilon(t, 4.8343e+06, initial, 0.01)
	assert.InEpsilon(t, 2.7766e+06, final, 0.01)
}
