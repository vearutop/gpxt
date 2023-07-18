// Package main provides GPX Tool binary.
package main

import (
	"errors"
	"fmt"
	"github.com/bool64/dev/version"
	"os"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/runnerup"
	_ "modernc.org/sqlite"
)

func main() {
	timeCmd()
	infoCmd()
	mergeCmd()
	runnerup.Cmd()

	kingpin.Version(version.Info().Version)
	kingpin.Parse()
}

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

		return nil
	})
}

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

func mergeCmd() {
	var (
		files  []string
		output string
		indent bool
	)

	merge := kingpin.Command("merge", "Merge multiple GPX tracks in one.")
	merge.Arg("files", "Files to merge.").StringsVar(&files)
	merge.Flag("output", "Output file.").Default("out.gpx").StringVar(&output)
	merge.Flag("indent", "Indent output file.").BoolVar(&indent)
	merge.Action(func(_ *kingpin.ParseContext) error {
		if len(files) < 2 {
			return errors.New("at least two files expected for merge")
		}

		gpxFile, err := gpx.ParseFile(files[0])
		if err != nil {
			return fmt.Errorf("error opening gpx file: %w", err)
		}

		for _, f := range files[1:] {
			mf, err := gpx.ParseFile(f)
			if err != nil {
				return fmt.Errorf("error opening gpx file: %w", err)
			}

			for _, t := range mf.Tracks {
				for _, s := range t.Segments {
					s := s
					gpxFile.AppendSegment(&s)
				}
			}
		}

		xx, err := gpxFile.ToXml(gpx.ToXmlParams{Indent: indent})
		if err != nil {
			return fmt.Errorf("error rendering GPX: %w", err)
		}

		if err = os.WriteFile(output, xx, 0o600); err != nil {
			return fmt.Errorf("failed to save GPX file: %w", err)
		}

		return nil
	})
}

// GetGpxElementInfo pretty prints some basic information about this GPX elements.
func GetGpxElementInfo(prefix string, gpxDoc gpx.GPXElementInfo) string {
	result := ""
	result += fmt.Sprint(prefix, " Points: ", gpxDoc.GetTrackPointsNo(), "\n")
	result += fmt.Sprint(prefix, " Length 2D: ", gpxDoc.Length2D()/1000.0, "km\n")
	result += fmt.Sprint(prefix, " Length 3D: ", gpxDoc.Length3D()/1000.0, "km\n")

	bounds := gpxDoc.Bounds()
	result += fmt.Sprintf("%s Bounds: %f, %f, %f, %f\n", prefix, bounds.MinLatitude, bounds.MaxLatitude, bounds.MinLongitude, bounds.MaxLongitude)

	md := gpxDoc.MovingData()
	result += fmt.Sprint(prefix, " Moving time: ", (time.Duration(md.MovingTime) * time.Second).String(), "\n")
	result += fmt.Sprint(prefix, " Stopped time: ", (time.Duration(md.StoppedTime) * time.Second).String(), "\n")

	result += fmt.Sprintf("%s Max speed: %fm/s = %fkm/h\n", prefix, md.MaxSpeed, md.MaxSpeed*60*60/1000.0)

	updo := gpxDoc.UphillDownhill()
	result += fmt.Sprint(prefix, " Total uphill: ", updo.Uphill, "\n")
	result += fmt.Sprint(prefix, " Total downhill: ", updo.Downhill, "\n")

	timeBounds := gpxDoc.TimeBounds()
	result += fmt.Sprint(prefix, " Started: ", timeBounds.StartTime.Format(time.RFC3339), "\n")
	result += fmt.Sprint(prefix, " Ended: ", timeBounds.EndTime.Format(time.RFC3339), "\n")

	return result
}
