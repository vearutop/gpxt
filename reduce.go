package main

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/tkrajina/gpxgo/gpx"
)

func reduceCmd() {
	var (
		files       []string
		minDist     int
		minInterval time.Duration
		output      string
		indent      bool
	)

	cmd := kingpin.Command("reduce", "Reduce number of points in track to simplify shape")
	cmd.Arg("files", "GPX files to process.").Required().StringsVar(&files)
	cmd.Flag("min-dist", "Min distance between points, meters.").Default("50").IntVar(&minDist)
	cmd.Flag("min-interval", "Min time interval between points, duration.").Default("30s").DurationVar(&minInterval)
	cmd.Flag("output", "Output file.").Default("<name>.reduced.gpx").StringVar(&output)
	cmd.Flag("indent", "Indent output file.").BoolVar(&indent)

	cmd.Action(func(_ *kingpin.ParseContext) error {
		for _, file := range files {
			gpxFile, err := gpx.ParseFile(file)
			if err != nil {
				return fmt.Errorf("error opening gpx file: %w", err)
			}

			for ti, t := range gpxFile.Tracks {
				for si, s := range t.Segments {
					s.Points = reducePoints(s.Points, minDist, minInterval)
					t.Segments[si] = s
				}

				gpxFile.Tracks[ti] = t
			}

			fmt.Println(GetGpxElementInfo("", gpxFile))

			xx, err := gpxFile.ToXml(gpx.ToXmlParams{Indent: indent})
			if err != nil {
				return fmt.Errorf("render GPX: %w", err)
			}

			name := strings.TrimSuffix(file, path.Ext(file))
			outName := strings.ReplaceAll(output, "<name>", name)

			if err = os.WriteFile(outName, xx, 0o600); err != nil {
				return fmt.Errorf("save GPX file: %w", err)
			}
		}

		return nil
	})
}

func reducePoints(points []gpx.GPXPoint, minDist int, minInterval time.Duration) []gpx.GPXPoint {
	var (
		prev gpx.GPXPoint
		pts  []gpx.GPXPoint
	)

	for i, pt := range points {
		if prev.Timestamp.IsZero() {
			prev = pt
			pts = append(pts, pt)

			continue
		}

		if prev.Distance2D(&pt) >= float64(minDist) {
			prev = pt
			pts = append(pts, pt)

			continue
		}

		if prev.Timestamp.Sub(pt.Timestamp) >= minInterval {
			prev = pt
			pts = append(pts, pt)

			continue
		}

		// Adding last point.
		if i == len(points)-1 {
			pts = append(pts, pt)
		}
	}

	return pts
}
