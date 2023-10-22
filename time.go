package main

import (
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/tkrajina/gpxgo/gpx"
)

func timeCmd() {
	var (
		file     string
		newStart string
		newEnd   string
		output   string
		indent   bool
	)

	cmd := kingpin.Command("time", "Move track in time")
	cmd.Arg("file", "GPX File to process.").Required().StringVar(&file)
	cmd.Flag("new-start", "New time of track start, e.g. 2022-05-28T10:36:34Z.").StringVar(&newStart)
	cmd.Flag("new-end", "New time of track end, e.g. 2022-05-28T10:36:34Z.").StringVar(&newEnd)
	cmd.Flag("output", "Output file.").Default("out.gpx").StringVar(&output)
	cmd.Flag("indent", "Indent output file.").BoolVar(&indent)

	cmd.Action(func(_ *kingpin.ParseContext) error {
		gpxFile, err := gpx.ParseFile(file)
		if err != nil {
			return fmt.Errorf("error opening gpx file: %w", err)
		}

		var delta time.Duration

		if newStart != "" {
			start, err := time.Parse(time.RFC3339, newStart)
			if err != nil {
				return fmt.Errorf("failed to parse new start time: %w", err)
			}

			delta = start.Sub(gpxFile.TimeBounds().StartTime)
		}

		if newEnd != "" {
			end, err := time.Parse(time.RFC3339, newEnd)
			if err != nil {
				return fmt.Errorf("failed to parse new end time: %w", err)
			}

			delta = end.Sub(gpxFile.TimeBounds().EndTime)
		}

		if delta == 0 {
			fmt.Println("no delta to apply")

			return nil
		}

		fmt.Println("delta to apply:", delta.String())

		gpxFile.ExecuteOnAllPoints(func(point *gpx.GPXPoint) {
			point.Timestamp = point.Timestamp.Add(delta)
		})

		fmt.Println(GetGpxElementInfo("", gpxFile))

		xx, err := gpxFile.ToXml(gpx.ToXmlParams{})
		if err != nil {
			return fmt.Errorf("error rendering GPX: %w", err)
		}

		if err = os.WriteFile(output, xx, 0o600); err != nil {
			return fmt.Errorf("failed to save GPX file: %w", err)
		}

		return nil
	})
}
