package main

import (
	"fmt"
	"os"
	"path"
	"strconv"
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
		durMul   float64
		output   string
		indent   bool
	)

	cmd := kingpin.Command("move", "Move track in time")
	cmd.Help("When both new-start and new-end are present, the track would be " +
		"\nstretched/shrunk to fit in new boundaries." +
		"\nOtherwise it would be moved to the touch new-start or new-end.")
	cmd.Arg("file", "GPX File to process.").Required().StringVar(&file)
	cmd.Flag("new-start", "New time of track start, e.g. 2022-05-28T10:36:34Z.").StringVar(&newStart)
	cmd.Flag("new-end", "New time of track end, e.g. 2022-05-28T10:36:34Z.").StringVar(&newEnd)
	cmd.Flag("dur-mul", "Duration multiplier, ignored if both new-start and new-end are present.").
		Default("1.0").Float64Var(&durMul)
	cmd.Flag("output", "Output file.").Default("<name>.moved.gpx").StringVar(&output)
	cmd.Flag("indent", "Indent output file.").BoolVar(&indent)

	cmd.Action(func(_ *kingpin.ParseContext) error {
		gpxFile, err := gpx.ParseFile(file)
		if err != nil {
			return fmt.Errorf("error opening gpx file: %w", err)
		}

		tt := timeTransformer{}
		tt.origMin = gpxFile.TimeBounds().StartTime
		tt.origDur = gpxFile.TimeBounds().EndTime.Sub(tt.origMin)

		if newStart != "" {
			start, err := time.Parse(time.RFC3339, newStart)
			if err != nil {
				return fmt.Errorf("failed to parse new start time: %w", err)
			}

			tt.newMin = start
			tt.delta = start.Sub(gpxFile.TimeBounds().StartTime)

			if newEnd == "" && durMul != 1.0 && durMul > 0 {
				tt.newDur = time.Duration(float64(tt.origDur) * durMul)
			}
		}

		if newEnd != "" {
			end, err := time.Parse(time.RFC3339, newEnd)
			if err != nil {
				return fmt.Errorf("failed to parse new end time: %w", err)
			}

			if newStart != "" {
				tt.newDur = end.Sub(tt.newMin)
			} else if durMul != 1.0 && durMul > 0 {
				tt.newDur = time.Duration(float64(tt.origDur) * durMul)
				tt.newMin = end.Add(-tt.newDur)
			}

			tt.delta = end.Sub(gpxFile.TimeBounds().EndTime)
		}

		if tt.newDur == 0 && durMul != 1.0 && durMul > 0 {
			tt.newDur = time.Duration(float64(tt.origDur) * durMul)
		}

		if tt.delta == 0 {
			fmt.Println("no delta to apply")

			return nil
		}

		fmt.Println("delta to apply:", tt.delta.String())

		if gpxFile.Name != "" {
			gpxFile.Name = "untitled1"
		}

		if gpxFile.Time != nil {
			tt.tr(gpxFile.Time)
		}

		for i, t := range gpxFile.Tracks {
			if t.Name != "" {
				t.Name = "track" + strconv.Itoa(i)
				gpxFile.Tracks[i] = t
			}
		}

		gpxFile.ExecuteOnAllPoints(func(point *gpx.GPXPoint) {
			tt.tr(&point.Timestamp)
		})

		fmt.Println(GetGpxElementInfo("", gpxFile))

		xx, err := gpxFile.ToXml(gpx.ToXmlParams{Indent: indent})
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

type timeTransformer struct {
	origMin time.Time
	origDur time.Duration
	newMin  time.Time
	newDur  time.Duration
	delta   time.Duration
}

func (tt timeTransformer) tr(
	p *time.Time,
) {
	if tt.newDur == 0 {
		*p = p.Add(tt.delta)

		return
	}

	pr := float64(p.Sub(tt.origMin)) / float64(tt.origDur)
	cr := pr * float64(tt.newDur) / float64(tt.origDur)
	*p = tt.newMin.Add(time.Duration(cr * float64(tt.newDur)))
}
