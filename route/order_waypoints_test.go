package route_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/route"
)

func TestOrderWaypoints(t *testing.T) {
	gpxFile, err := gpx.ParseFile("testdata/photos-2023-10-faro.gpx")
	require.NoError(t, err)

	// Evolution completed
	// Initial tour distance:  4.92231483363616e+06
	// Final tour distance:  2.796963430731682e+06

	route.OrderWaypoints(gpxFile.Waypoints)
}

func TestOrderWaypoints_small(t *testing.T) {
	gpxFile, err := gpx.ParseFile("testdata/photos-2023-10-faro.gpx")
	require.NoError(t, err)

	// Evolution completed
	// Initial tour distance:  3.4733577288727323e+06
	// Final tour distance:  1.3693686948976656e+06

	// Evolution completed
	// Initial tour distance:  36587.45264918967
	// Final tour distance:  12734.230975153981

	route.OrderWaypoints(gpxFile.Waypoints[0:30])
}
