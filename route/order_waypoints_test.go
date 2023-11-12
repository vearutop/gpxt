package route_test

import (
	"github.com/stretchr/testify/require"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/route"
	"testing"
)

func TestOrderWaypoints(t *testing.T) {
	gpxFile, err := gpx.ParseFile("testdata/photos-2023-10-faro.gpx")
	require.NoError(t, err)

	route.OrderWaypoints(gpxFile.Waypoints)
}

func TestOrderWaypoints_small(t *testing.T) {
	gpxFile, err := gpx.ParseFile("testdata/photos-2023-10-faro.gpx")
	require.NoError(t, err)

	// Evolution completed
	//Initial tour distance:  3.4733577288727323e+06
	//Final tour distance:  1.3693686948976656e+06

	route.OrderWaypoints(gpxFile.Waypoints[0:30])
}
