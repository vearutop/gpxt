package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/vearutop/gpxt/route/tsp"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/route/ors"
)

func routeCmd() {
	var (
		file      string
		output    string
		detailed  bool
		profile   string
		indent    bool
		gens, pop int
	)

	cmd := kingpin.Command("route", "Build optimal route through waypoints")
	cmd.Arg("file", "GPX files to process.").Required().StringVar(&file)
	cmd.Flag("gens", "Number of generations for genetic algorithm.").Default(strconv.Itoa(tsp.DefaultNumberOfGenerations)).IntVar(&gens)
	cmd.Flag("pop", "Population size for genetic algorithm.").Default(strconv.Itoa(tsp.DefaultPopulationSize)).IntVar(&pop)
	cmd.Flag("profile", "Roting profile for ORS.").Default(string(ors.ProfileFootWalking)).StringVar(&profile)
	cmd.Flag("detailed", "Add routing directions from api.openrouteservice.org (needs api key in ORS_KEY env var)").BoolVar(&detailed)
	cmd.Flag("output", "Output file.").Default("<name>.routed.gpx").StringVar(&output)
	cmd.Flag("indent", "Indent output file.").BoolVar(&indent)

	cmd.Action(func(_ *kingpin.ParseContext) error {
		ctx := context.Background()

		gpxFile, err := gpx.ParseFile(file)
		if err != nil {
			return fmt.Errorf("error opening gpx file: %w", err)
		}

		if len(gpxFile.Waypoints) == 0 {
			return errors.New("no waypoints in gpx file")
		}

		points, initial, final := tsp.Order(gpxFile.Waypoints, gens, pop)

		fmt.Printf("Points graph ordered, initial distance: %.2f, final distance: %.2f\n", initial, final)

		if detailed {
			gj, err := ors.GetRoute(ctx, ors.Profile(profile), points)
			if err != nil {
				return fmt.Errorf("get routing directions: %w", err)
			}

			segs := gj.Features[0].Properties.Segments
			pts := gj.Features[0].Geometry.Coordinates
			totalDist := 0.0
			var totalDur time.Duration

			for _, seg := range segs {
				t := gpx.GPXTrack{}
				totalDist += seg.Distance
				dur := time.Duration(seg.Duration * float64(time.Second))
				totalDur += dur
				t.Description = fmt.Sprintf("%.2fm, %s\n", seg.Distance, dur.String())

				s := gpx.GPXTrackSegment{}

				start := seg.Steps[0]
				end := seg.Steps[len(seg.Steps)-1]

				for i := start.WayPoints[0]; i <= end.WayPoints[1]; i++ {
					p := gpx.GPXPoint{}
					pt := pts[i]
					p.Longitude = pt[0]
					p.Latitude = pt[1]
					s.AppendPoint(&p)
				}

				t.AppendSegment(&s)
				gpxFile.AppendTrack(&t)
			}

			fmt.Printf("Total detailed distance: %.2f, total time %s\n", totalDist, totalDur.String())
		} else {
			var pts [][2]float64

			for _, p := range points {
				gp := gpx.GPXPoint{}
				gp.Point = p
				gpxFile.AppendPoint(&gp)
				pts = append(pts, [2]float64{p.Longitude, p.Latitude})
			}

			j, _ := json.Marshal(pts)
			fmt.Println(string(j))
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

		return nil
	})
}
