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

func cutCmd() {
	var (
		files   []string
		output  string
		indent  bool
		minTime string
		maxTime string
	)

	cmd := kingpin.Command("cut", "Remove head and/or tail of a track")
	cmd.Arg("files", "GPX files to process.").Required().StringsVar(&files)
	cmd.Flag("min-time", "Min allowed timestamp, e.g. 2022-05-28T10:36:34Z.").StringVar(&minTime)
	cmd.Flag("max-time", "Max allowed timestamp, e.g. 2022-05-28T10:36:34Z.").StringVar(&maxTime)
	cmd.Flag("output", "Output file.").Default("<name>.cut.gpx").StringVar(&output)
	cmd.Flag("indent", "Indent output file.").BoolVar(&indent)

	cmd.Action(func(_ *kingpin.ParseContext) error {
		minTS, err := time.Parse(time.RFC3339, minTime)
		if err != nil && minTime != "" {
			return fmt.Errorf("parsing min-time: %w", err)
		}

		maxTS, err := time.Parse(time.RFC3339, maxTime)
		if err != nil && maxTime != "" {
			return fmt.Errorf("parsing max-time: %w", err)
		}

		for _, file := range files {
			gpxFile, err := gpx.ParseFile(file)
			if err != nil {
				return fmt.Errorf("error opening gpx file: %w", err)
			}

			for ti, t := range gpxFile.Tracks {
				for si, s := range t.Segments {
					s.Points = cutPoints(s.Points, minTS, maxTS)
					t.Segments[si] = s
				}

				gpxFile.Tracks[ti] = t
			}

			fmt.Println(GetGpxElementInfo("", gpxFile))

			xx, err := gpxFile.ToXml(gpx.ToXmlParams{})
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

func cutPoints(points []gpx.GPXPoint, minTime, maxTime time.Time) []gpx.GPXPoint {
	var (
		pts []gpx.GPXPoint
	)

	for _, pt := range points {
		pt := pt

		if !minTime.IsZero() && pt.Timestamp.Before(minTime) {
			continue
		}

		if !maxTime.IsZero() && pt.Timestamp.After(minTime) {
			continue
		}

		pts = append(pts, pt)
	}

	return pts
}
