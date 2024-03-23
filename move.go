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

func moveCmd() {
	var (
		file     string
		newStart string
		newEnd   string
		output   string
		indent   bool
	)

	cmd := kingpin.Command("move", "Move track in time")
	cmd.Help("When both new-start and new-end are present, the track would be " +
		"\nstretched/shrinked to fit in new boundaries." +
		"\nOtherwise it would be moved to the touch new-start or new-end.")
	cmd.Arg("file", "GPX File to process.").Required().StringVar(&file)
	cmd.Flag("new-start", "New time of track start, e.g. 2022-05-28T10:36:34Z.").StringVar(&newStart)
	cmd.Flag("new-end", "New time of track end, e.g. 2022-05-28T10:36:34Z.").StringVar(&newEnd)
	cmd.Flag("output", "Output file.").Default("<name>.cut.gpx").StringVar(&output)
	cmd.Flag("indent", "Indent output file.").BoolVar(&indent)

	cmd.Action(func(_ *kingpin.ParseContext) error {
		gpxFile, err := gpx.ParseFile(file)
		if err != nil {
			return fmt.Errorf("error opening gpx file: %w", err)
		}

		var (
			origMin = gpxFile.TimeBounds().StartTime
			origDur = gpxFile.TimeBounds().EndTime.Sub(origMin)

			newMin time.Time
			newDur time.Duration
		)

		var delta time.Duration

		if newStart != "" {
			start, err := time.Parse(time.RFC3339, newStart)
			if err != nil {
				return fmt.Errorf("failed to parse new start time: %w", err)
			}

			newMin = start
			delta = start.Sub(gpxFile.TimeBounds().StartTime)
		}

		if newEnd != "" {
			end, err := time.Parse(time.RFC3339, newEnd)
			if err != nil {
				return fmt.Errorf("failed to parse new end time: %w", err)
			}

			if !newMin.IsZero() {
				newDur = end.Sub(newMin)
			}

			delta = end.Sub(gpxFile.TimeBounds().EndTime)
		}

		if delta == 0 {
			fmt.Println("no delta to apply")

			return nil
		}

		fmt.Println("delta to apply:", delta.String())

		gpxFile.ExecuteOnAllPoints(func(point *gpx.GPXPoint) {
			if newDur != 0 {
				point.Timestamp = transformTime(origMin, origDur, newMin, newDur, point.Timestamp)
			} else {
				point.Timestamp = point.Timestamp.Add(delta)
			}
		})

		fmt.Println(GetGpxElementInfo("", gpxFile))

		xx, err := gpxFile.ToXml(gpx.ToXmlParams{})
		if err != nil {
			return fmt.Errorf("error rendering GPX: %w", err)
		}

		name := strings.TrimSuffix(file, path.Ext(file))
		outName := strings.ReplaceAll(output, "<name>", name)

		if err = os.WriteFile(outName, xx, 0o600); err != nil {
			return fmt.Errorf("failed to save GPX file: %w", err)
		}

		return nil
	})
}

func transformTime(
	origMin time.Time,
	origDur time.Duration,
	newMin time.Time,
	newDur time.Duration,
	p time.Time) time.Time {
	pr := float64(p.Sub(origMin)) / float64(origDur)
	cr := pr * float64(newDur) / float64(origDur)
	c := newMin.Add(time.Duration(cr * float64(newDur)))

	return c
}
