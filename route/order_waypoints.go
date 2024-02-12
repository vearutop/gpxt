package route

import (
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/route/tsp"
)

func OrderWaypoints(waypoints []gpx.GPXPoint) []gpx.Point {
	return tsp.Order(waypoints)
}
