package main

import (
	"fmt"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/speedstat"
)

func infoCmd() {
	var (
		file string

		speedHist        bool
		speedBin         float64
		speedMinInterval time.Duration
		speedMinDist     float64
		speedMaxHDOP     float64
		speedMaxKmh      float64
	)

	info := kingpin.Command("info", "Show info about GPX file")
	info.Arg("file", "File to show info for.").Required().StringVar(&file)
	info.Flag("speed-hist", "Show speed histogram with distance/time percentiles.").BoolVar(&speedHist)
	info.Flag("speed-bin", "Speed histogram bucket width, km/h.").Default("5").Float64Var(&speedBin)
	info.Flag("speed-min-interval", "Minimum duration of a speed sample, shorter gaps are merged "+
		"into the next sample to reduce GPS jitter noise.").Default("3s").DurationVar(&speedMinInterval)
	info.Flag("speed-min-dist", "Minimum distance of a speed sample in meters, shorter gaps are merged "+
		"into the next sample to reduce GPS jitter noise.").Default("15").Float64Var(&speedMinDist)
	info.Flag("speed-max-hdop", "Ignore points with horizontal dilution of precision above this value, "+
		"0 disables the check.").Default("0").Float64Var(&speedMaxHDOP)
	info.Flag("speed-max-kmh", "Exclude samples faster than this as GPS noise, "+
		"0 disables the check.").Default("0").Float64Var(&speedMaxKmh)
	info.Action(func(_ *kingpin.ParseContext) error {
		gpxFile, err := gpx.ParseFile(file)
		if err != nil {
			return fmt.Errorf("error opening gpx file: %w", err)
		}

		fmt.Println(GetGpxElementInfo("", gpxFile))

		if speedHist {
			opt := speedstat.Options{
				MinInterval: speedMinInterval,
				MinDistance: speedMinDist,
				MaxHDOP:     speedMaxHDOP,
				MaxSpeedKmh: speedMaxKmh,
				BinWidth:    speedBin,
			}

			fmt.Println(SpeedHistReport(gpxFile, opt))
		}

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
				for _, point := range s.Points {
					if prevPoint != nil {
						dist += prevPoint.Distance2D(&point)
					}

					prevPoint = &point
				}
			}
		}

		totalGPXDist := dist
		fmt.Printf("GPX dist 2: %.2fkm\n", totalGPXDist/1000.0)

		return nil
	})
}
