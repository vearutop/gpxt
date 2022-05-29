package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/tkrajina/gpxgo/gpx"
)

func main() {
	flag.Parse()

	app := kingpin.New("gpxt", "Hello!")

	args := flag.Args()
	if len(args) != 1 {
		fmt.Println("Please provide a GPX file path!")
		return
	}

	gpxFileArg := args[0]
	gpxFile, err := gpx.ParseFile(gpxFileArg)
	if err != nil {
		fmt.Println("Error opening gpx file: ", err)
		return
	}

	gpxPath, _ := filepath.Abs(gpxFileArg)

	fmt.Print("File: ", gpxPath, "\n")

	start := gpxFile.Tracks[0].TimeBounds().StartTime

	newStart := time.Now().Add(-time.Hour - 50*time.Minute).UTC()
	println(start.String(), newStart.String())

	delta := newStart.Sub(start)

	gpxFile.ExecuteOnAllPoints(func(point *gpx.GPXPoint) {
		point.Timestamp = point.Timestamp.Add(delta)
	})

	xx, err := gpxFile.ToXml(gpx.ToXmlParams{})
	if err != nil {
		log.Fatal(err)
	}

	ioutil.WriteFile("./out.gpx", xx, 0o660)

	fmt.Println(gpxFile.GetGpxInfo())
}
