package main

import (
	"fmt"

	"github.com/alecthomas/kingpin/v2"
	"github.com/tkrajina/gpxgo/gpx"
)

func infoCmd() {
	var file string

	info := kingpin.Command("info", "Show info about GPX file")
	info.Arg("file", "File to show info for.").Required().StringVar(&file)
	info.Action(func(_ *kingpin.ParseContext) error {
		gpxFile, err := gpx.ParseFile(file)
		if err != nil {
			return fmt.Errorf("error opening gpx file: %w", err)
		}

		fmt.Println(GetGpxElementInfo("", gpxFile))

		if len(gpxFile.Tracks) > 0 {
			fmt.Println("Tracks:", len(gpxFile.Tracks))

			for i, t := range gpxFile.Tracks {
				fmt.Println("Track", i+1, "segments:", len(t.Segments))
			}
		}

		if len(gpxFile.Waypoints) > 0 {
			fmt.Println("Waypoints:", len(gpxFile.Waypoints))
		}

		if len(gpxFile.Routes) > 0 {
			fmt.Println("Routes:", len(gpxFile.Routes))
		}

		var (
			dist      float64
			prevPoint *gpx.GPXPoint
		)

		for _, tr := range gpxFile.Tracks {
			for _, s := range tr.Segments {
				for _, point := range s.Points {
					if prevPoint != nil {
						dist += prevPoint.Distance2D(&point)
					}

					prevPoint = &point
				}
			}
		}

		totalGPXDist := dist
		fmt.Printf("GPX dist 2: %.2fkm\n", totalGPXDist/1000.0)

		return nil
	})
}
