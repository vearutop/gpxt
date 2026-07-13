package speedstat_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/speedstat"
)

func pt(t time.Time, lat, lon float64, hdop float64) gpx.GPXPoint {
	p := gpx.GPXPoint{
		Point: gpx.Point{
			Latitude:  lat,
			Longitude: lon,
		},
		Timestamp: t,
	}

	if hdop > 0 {
		p.HorizontalDilution.SetValue(hdop)
	}

	return p
}

func TestSamples_mergesShortNoisyIntervals(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Steady ~10km/h walk: one point per second, moving ~2.78m/s.
	var pts []gpx.GPXPoint

	lat := 50.0

	for i := range 20 {
		pts = append(pts, pt(base.Add(time.Duration(i)*time.Second), lat, 0, 0))
		lat += 0.000025 // roughly 2.78m per step near the equator-ish latitude used here
	}

	gpxFile := &gpx.GPX{Tracks: []gpx.GPXTrack{{Segments: []gpx.GPXTrackSegment{{Points: pts}}}}}

	samples := speedstat.Samples(gpxFile, speedstat.Options{
		MinInterval: 3 * time.Second,
		MinDistance: 5,
	})

	require.NotEmpty(t, samples)

	var totalDist, totalDur float64

	for _, s := range samples {
		assert.GreaterOrEqual(t, s.Duration, 3.0, "each sample should respect MinInterval, except possibly a merged tail")
		totalDist += s.Distance
		totalDur += s.Duration
	}

	assert.InDelta(t, 19*time.Second.Seconds(), totalDur, 0.001, "no time should be lost to merging")
	assert.Greater(t, totalDist, 0.0)
}

func TestSamples_skipsBadHDOPPoints(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	pts := []gpx.GPXPoint{
		pt(base, 50.0, 0, 1),
		pt(base.Add(1*time.Second), 55.0, 0, 20), // wild jump due to bad fix, high HDOP
		pt(base.Add(2*time.Second), 50.00006, 0, 1),
	}

	gpxFile := &gpx.GPX{Tracks: []gpx.GPXTrack{{Segments: []gpx.GPXTrackSegment{{Points: pts}}}}}

	samples := speedstat.Samples(gpxFile, speedstat.Options{MaxHDOP: 5})

	require.Len(t, samples, 1)
	assert.Less(t, samples[0].SpeedKmh, 50.0, "bridging over the bad fix should avoid the spurious spike")
}

func TestNewHistogram_excludesImplausibleSpeeds(t *testing.T) {
	samples := []speedstat.Sample{
		{SpeedKmh: 10, Distance: 100, Duration: 36},
		{SpeedKmh: 500, Distance: 50, Duration: 0.36}, // GPS noise spike
	}

	h := speedstat.NewHistogram(samples, speedstat.Options{MaxSpeedKmh: 100})

	assert.Len(t, h.Samples, 1)
	assert.Equal(t, 1, h.Excluded.Count)
	assert.InDelta(t, 50.0, h.Excluded.Distance, 0.001)
	assert.InDelta(t, 100.0, h.TotalDistance, 0.001)
}

func TestNewHistogram_buckets(t *testing.T) {
	samples := []speedstat.Sample{
		{SpeedKmh: 2, Distance: 10, Duration: 18},
		{SpeedKmh: 7, Distance: 20, Duration: 10},
		{SpeedKmh: 12, Distance: 30, Duration: 9},
	}

	h := speedstat.NewHistogram(samples, speedstat.Options{BinWidth: 5})

	require.Len(t, h.Buckets, 3)
	assert.Equal(t, speedstat.Bucket{FromKmh: 0, ToKmh: 5, Count: 1, Distance: 10, Duration: 18}, h.Buckets[0])
	assert.Equal(t, speedstat.Bucket{FromKmh: 5, ToKmh: 10, Count: 1, Distance: 20, Duration: 10}, h.Buckets[1])
	assert.Equal(t, speedstat.Bucket{FromKmh: 10, ToKmh: 15, Count: 1, Distance: 30, Duration: 9}, h.Buckets[2])
}

func TestHistogram_SpeedAbove(t *testing.T) {
	// 90 meters at 5km/h, 10 meters at 50km/h: 90% of distance is at the slow speed.
	samples := []speedstat.Sample{
		{SpeedKmh: 5, Distance: 90, Duration: 64.8},
		{SpeedKmh: 50, Distance: 10, Duration: 0.72},
	}

	h := speedstat.NewHistogram(samples, speedstat.Options{})

	// 90% of distance should be covered at or above ~5km/h (everything, since only the
	// top 10% - the fast sample - is excluded from the "above" threshold).
	assert.InDelta(t, 5.0, h.SpeedAbove(90, speedstat.ByDistance), 0.5)

	// Asking for coverage beyond the slow bulk should push the threshold toward the fast sample.
	assert.Greater(t, h.SpeedAbove(5, speedstat.ByDistance), 5.0)
}

func TestHistogram_SpeedAbove_empty(t *testing.T) {
	h := speedstat.NewHistogram(nil, speedstat.Options{})
	assert.Equal(t, 0.0, h.SpeedAbove(90, speedstat.ByDistance))
}
