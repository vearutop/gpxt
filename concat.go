package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/tkrajina/gpxgo/gpx"
)

func concatCmd() {
	var (
		files  []string
		output string
		indent bool
	)

	merge := kingpin.Command("concat", "Concat multiple GPX tracks in one")
	merge.Arg("files", "Files to merge.").StringsVar(&files)
	merge.Flag("output", "Output file.").Default("out.gpx").StringVar(&output)
	merge.Flag("indent", "Indent output file.").BoolVar(&indent)
	merge.Action(func(_ *kingpin.ParseContext) error {
		if len(files) < 2 {
			return errors.New("at least two files expected for merge")
		}

		gpxFile, err := gpx.ParseFile(files[0])
		if err != nil {
			return fmt.Errorf("open gpx file: %w", err)
		}

		tm := gpxFile.TimeBounds().EndTime

		for _, f := range files[1:] {
			mf, err := gpx.ParseFile(f)
			if err != nil {
				return fmt.Errorf("open gpx file: %w", err)
			}

			for _, t := range mf.Tracks {
				for _, s := range t.Segments {
					tb := s.TimeBounds()
					dt := tb.StartTime.Unix() - tm.Unix()

					if dt > 0 && dt < 300 { // Up to 5 min.
						gpxFile.AppendSegment(&s)

						tm = tb.EndTime
					} else {
						dt -= 60
						for _, p := range s.Points {
							p.Timestamp = p.Timestamp.Add(-time.Duration(dt) * time.Second)
							tm = p.Timestamp
							gpxFile.AppendPoint(&p)
						}
					}
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
