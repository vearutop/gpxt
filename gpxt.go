// Package main provides GPX Tool binary.
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/bool64/dev/version"
	"github.com/vearutop/gpxt/gpx"
	"github.com/vearutop/gpxt/runnerup"
	"github.com/vearutop/gpxt/sigma"
	_ "modernc.org/sqlite"
)

func main() {
	moveCmd()
	infoCmd()
	showCmd()
	concatCmd()
	cutCmd()
	reduceCmd()
	routeCmd()
	runnerup.Cmd()
	sigma.Cmd()

	kingpin.Version(version.Info().Version)
	kingpin.Parse()
}

// GetGpxElementInfo pretty prints some basic information about this GPX elements.
func GetGpxElementInfo(prefix string, gpxDoc gpx.GPXElementInfo) string {
	result := ""
	result += fmt.Sprint(prefix, "Points: ", gpxDoc.GetTrackPointsNo(), "\n")
	result += fmt.Sprint(prefix, "Length 2D: ", gpxDoc.Length2D()/1000.0, "km\n")
	result += fmt.Sprint(prefix, "Length 3D: ", gpxDoc.Length3D()/1000.0, "km\n")

	bounds := gpxDoc.Bounds()
	result += strings.TrimSpace(fmt.Sprintf("%s Bounds: %f, %f, %f, %f",
		prefix, bounds.MinLatitude, bounds.MaxLatitude, bounds.MinLongitude,
		bounds.MaxLongitude)) + "\n"

	md := gpxDoc.MovingData()
	result += fmt.Sprint(prefix, "Moving time: ", (time.Duration(md.MovingTime) * time.Second).String(), "\n")
	result += fmt.Sprint(prefix, "Stopped time: ", (time.Duration(md.StoppedTime) * time.Second).String(), "\n")

	avgSpd := md.MovingDistance / md.MovingTime // m/s

	result += strings.TrimSpace(fmt.Sprintf("%s Avg speed: %fm/s = %fkm/h",
		prefix, avgSpd, avgSpd*60*60/1000.0)) + "\n"

	result += strings.TrimSpace(fmt.Sprintf("%s Max speed: %fm/s = %fkm/h",
		prefix, md.MaxSpeed, md.MaxSpeed*60*60/1000.0)) + "\n"

	updo := gpxDoc.UphillDownhill()
	result += fmt.Sprint(prefix, "Total uphill: ", updo.Uphill, "m\n")
	result += fmt.Sprint(prefix, "Total downhill: ", updo.Downhill, "m\n")

	timeBounds := gpxDoc.TimeBounds()
	result += fmt.Sprint(prefix, "Started: ", timeBounds.StartTime.Format(time.RFC3339), "\n")
	result += fmt.Sprint(prefix, "Ended: ", timeBounds.EndTime.Format(time.RFC3339), "\n")

	return result
}
