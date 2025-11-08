package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/tkrajina/gpxgo/gpx"
)

func extendCmd() {
	var (
		file   string
		factor float64
		output string
		indent bool
	)

	extend := kingpin.Command("extend", "Add jitter to make track longer or shorter")
	extend.Arg("file", "File to extend.").Required().StringVar(&file)
	extend.Flag("factor", "A float multiplier for lat/lon diff with prev point, use negative value to shrink.").Default("0.1").Float64Var(&factor)
	extend.Flag("output", "Output file.").Default("<name>.extended.gpx").StringVar(&output)
	extend.Flag("indent", "Indent output file.").BoolVar(&indent)

	extend.Action(func(_ *kingpin.ParseContext) error {
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
				for i, point := range s.Points {
					if prevPoint != nil {
						dlat := factor * (prevPoint.Latitude - point.Latitude)
						dlon := factor * (prevPoint.Longitude - point.Longitude)

						point.Latitude -= dlat
						point.Longitude -= dlon

						dist += prevPoint.Distance2D(&point)
					}

					s.Points[i] = point
					prevPoint = &point
				}
			}
		}

		totalGPXDist := dist
		fmt.Printf("GPX dist 2: %.2fkm\n", totalGPXDist/1000.0)

		xx, err := gpxFile.ToXml(gpx.ToXmlParams{Indent: indent})
		if err != nil {
			return fmt.Errorf("render GPX: %w", err)
		}

		name := strings.TrimSuffix(file, path.Ext(file))
		outName := strings.ReplaceAll(output, "<name>", name)

		if err = os.WriteFile(outName, xx, 0o600); err != nil {
			return fmt.Errorf("save GPX file: %w", err)
		}

		return nil
	})
}
