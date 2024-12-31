package runnerup

import (
	"context"
	"fmt"
	"os"

	"github.com/alecthomas/kingpin/v2"
	"github.com/vearutop/gpxt/gpx"
)

func export(c *kingpin.CmdClause) {
	var (
		dbFile     string
		activityID int
		output     string
		indent     bool
	)

	export := c.Command("export", "Export activity as GPX.")
	export.Arg("db", "RunnerUp DB export SQLite file.").Required().StringVar(&dbFile)
	export.Arg("activity-id", "Activity ID.").Required().IntVar(&activityID)
	export.Arg("output", "Output file (default <activity-id.gpx>).").StringVar(&output)
	export.Flag("indent", "indent output file.").BoolVar(&indent)

	export.Action(func(_ *kingpin.ParseContext) error {
		r, err := NewRepository(dbFile)
		if err != nil {
			return err
		}

		gpxDoc := gpx.GPX{}

		l, err := r.ListLocations(context.Background(), activityID)
		if err != nil {
			return err
		}

		for _, i := range l {
			p := gpx.GPXPoint{}
			p.Latitude = i.Lat
			p.Longitude = i.Lon
			p.Elevation.SetValue(i.Alt)

			if i.Satellites != 0 {
				p.Satellites.SetValue(i.Satellites)
			}

			p.Timestamp = i.Time.Time()

			gpxDoc.AppendPoint(&p)
		}

		xx, err := gpxDoc.ToXml(gpx.ToXmlParams{Indent: indent})
		if err != nil {
			return err
		}

		if output == "" {
			output = fmt.Sprintf("%d.xml", activityID)
		}

		return os.WriteFile(output, xx, 0o600)
	})
}
