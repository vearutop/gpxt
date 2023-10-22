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

		return nil
	})
}
