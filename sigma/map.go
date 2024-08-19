package sigma

import (
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/tkrajina/gpxgo/gpx"
)

// MapSlf defines mapping options.
type MapSlf struct {
	ByDist    bool
	FromStart bool
	Scale     float64
}

const (
	tpxNs   = "http://www.garmin.com/xmlschemas/TrackPointExtension/v1"
	tpxPath = "TrackPointExtension"
)

// MergeSlfIntoGpx adds data from SLF into GPX file.
func MergeSlfIntoGpx(slfFn, gpxFn, outFn string, opts ...func(options *MapSlf)) error {
	var v Activity

	mo := MapSlf{}

	for _, opt := range opts {
		opt(&mo)
	}

	d, err := os.ReadFile(slfFn) //nolint:gosec
	if err != nil {
		return fmt.Errorf("read source slf: %w", err)
	}

	if err := xml.Unmarshal(d, &v); err != nil {
		return fmt.Errorf("decode slf: %w", err)
	}

	entries := v.Entries.Entry

	gpxFile, err := gpx.ParseFile(gpxFn)
	if err != nil {
		return fmt.Errorf("parse source gpx: %w", err)
	}

	// Thu Aug 1 17:56:21 GMT+0200 2024
	slfStartTime, err := time.Parse("Mon Jan _2 15:04:05 GMT-0700 2006", v.GeneralInformation.StartDate)
	if err != nil {
		return err
	}

	entryTimePause := func(entry Entry) (time.Time, time.Duration) {
		s := time.Second * time.Duration(entry.TrainingTimeAbsolute/100)
		ts := slfStartTime.Add(s)

		var p time.Duration

		// Subtract pauses.
		for _, m := range v.Markers.Marker {
			if m.Type != "p" {
				continue
			}

			if entry.TrainingTimeAbsolute < m.TimeAbsolute {
				break
			}

			d := time.Second * time.Duration(m.Duration/100)
			p += d

			ts = ts.Add(d)
		}

		return ts, p
	}

	entryTime := func(entry Entry) time.Time {
		t, _ := entryTimePause(entry)

		return t
	}

	var (
		slfFullDist = entries[len(entries)-1].DistanceAbsolute
		dist        float64
		prevPoint   *gpx.GPXPoint
	)

	if !mo.FromStart {
		dist = gpxFile.Length2D()
	}

	visitPoint := func(point *gpx.GPXPoint) {
		if prevPoint != nil {
			dist += prevPoint.Distance2D(point)
		}

		prevPoint = point

		// Find closes point by time or by distance.
		i, _ := slices.BinarySearchFunc(entries, point, func(entry Entry, point *gpx.GPXPoint) int {
			if mo.ByDist {
				if entry.DistanceAbsolute < dist {
					return -1
				}

				return 1
			}

			ts := entryTime(entry)

			if ts.Before(point.Timestamp) {
				return -1
			}

			return 1
		})

		t := point.Timestamp

		if i < len(v.Entries.Entry) {
			vv := v.Entries.Entry[i]

			vt, _ := entryTimePause(vv)

			if !mo.ByDist && vt.Sub(t) > 10*time.Second {
				return
			}

			if mo.ByDist && math.Abs(vv.DistanceAbsolute-dist) > 100.0 {
				return
			}

			if vv.Power != nil {
				point.Extensions.GetOrCreateNode(gpx.NoNamespace, "power").Data = strconv.Itoa(int(*vv.Power))
			}

			if vv.Heartrate != nil && *vv.Heartrate != 0 {
				point.Extensions.GetOrCreateNode(tpxNs, tpxPath, "hr").Data = strconv.Itoa(int(*vv.Heartrate))
			}

			if vv.Cadence != nil {
				point.Extensions.GetOrCreateNode(tpxNs, tpxPath, "cad").Data = strconv.Itoa(int(*vv.Cadence))
			}

			if vv.Temperature != "" {
				point.Extensions.GetOrCreateNode(tpxNs, tpxPath, "atemp").Data = vv.Temperature
			}
		}
	}

	for _, tr := range gpxFile.Tracks {
		for _, s := range tr.Segments {
			prevPoint = nil

			for i, point := range s.Points {
				visitPoint(&point)
				s.Points[i] = point
			}
		}
	}

	xx, err := gpxFile.ToXml(gpx.ToXmlParams{Indent: true})
	if err != nil {
		return err
	}

	if err = os.WriteFile(outFn, xx, 0o600); err != nil {
		return err
	}

	return nil
}
