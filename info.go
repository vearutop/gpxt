package main

import (
	"fmt"

	"github.com/alecthomas/kingpin/v2"
	"github.com/vearutop/gpxt/gpx"
)

func infoCmd() {
	var (
		file     string
		segments bool
	)

	info := kingpin.Command("info", "Show info about GPX file")
	info.Arg("file", "File to show info for.").Required().StringVar(&file)
	info.Flag("segments", "Show details for segments.").BoolVar(&segments)
	info.Action(func(_ *kingpin.ParseContext) error {
		gpxFile, err := gpx.ParseFile(file)
		if err != nil {
			return fmt.Errorf("error opening gpx file: %w", err)
		}

		i := gpx.GetInfo(gpxFile)
		i.ShowSegments = segments

		fmt.Println(i.String())

		return nil
	})
}
