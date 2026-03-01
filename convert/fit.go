// Package convert provides tools to import bespoke GPX flavors.
package convert

import (
	"bytes"
	"fmt"
	"math"
	"strconv"

	"github.com/tkrajina/gpxgo/gpx"
	"github.com/tormoder/fit"
	"github.com/vearutop/gpxt/ns"
)

const (
	fitUint8Invalid  = 0xFF
	fitUint16Invalid = 0xFFFF
)

// FromFit reads Garmin FIT and returns GPX.
func FromFit(data []byte) (gpx.GPX, error) {
	var result gpx.GPX
	result.RegisterNamespace("gpxtpx", ns.TpxNs)

	file, err := fit.Decode(bytes.NewReader(data))
	if err != nil {
		return result, err
	}

	activity, err := file.Activity()
	if err != nil {
		return result, err
	}

	var firstTimestampSet bool
	for _, record := range activity.Records {
		if record.PositionLat.Invalid() || record.PositionLong.Invalid() {
			continue
		}

		p := gpx.GPXPoint{
			Point: gpx.Point{
				Latitude:  record.PositionLat.Degrees(),
				Longitude: record.PositionLong.Degrees(),
			},
		}

		if !record.Timestamp.IsZero() {
			p.Timestamp = record.Timestamp
			if !firstTimestampSet {
				ts := record.Timestamp
				result.Time = &ts
				firstTimestampSet = true
			}
		}

		alt := record.GetEnhancedAltitudeScaled()
		if math.IsNaN(alt) {
			alt = record.GetAltitudeScaled()
		}
		if !math.IsNaN(alt) {
			p.Elevation = *gpx.NewNullableFloat64(compactFloat(alt))
		}

		if record.Cadence != fitUint8Invalid {
			p.Extensions.GetOrCreateNode(ns.TpxNs, ns.TpxPath, "cad").Data = strconv.Itoa(int(record.Cadence))
		}
		if record.Power != fitUint16Invalid {
			p.Extensions.GetOrCreateNode(gpx.NoNamespace, "power").Data = strconv.Itoa(int(record.Power))
		}

		result.AppendPoint(&p)
	}

	if result.GetTrackPointsNo() == 0 {
		return result, fmt.Errorf("fit: no track points with position")
	}

	return result, nil
}

func compactFloat(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return value
	}

	rounded2 := math.Round(value*100) / 100
	if math.Abs(value-rounded2) < 1e-6 {
		return rounded2
	}

	return value
}
