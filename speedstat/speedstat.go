// Package speedstat computes speed distribution statistics (histogram and
// weighted percentiles) over a GPX track.
//
// Raw point-to-point speed is noisy at poor GPS signal: short time/distance
// deltas between jittery fixes produce implausible spikes. Samples merges
// consecutive points until both a minimum duration and a minimum distance are
// reached before trusting their speed, optionally skipping points with a bad
// HDOP fix. This keeps every meter and every second accounted for (merged,
// never dropped), so distance/time totals stay exact and percentages computed
// from the histogram remain meaningful.
package speedstat

import (
	"sort"
	"time"

	"github.com/tkrajina/gpxgo/gpx"
)

// Options configures how raw points are turned into speed Samples and how
// Samples are bucketed into a Histogram.
type Options struct {
	// MinInterval is the minimum duration a sample must span before its speed
	// is trusted. Shorter spans are merged into the following sample.
	MinInterval time.Duration

	// MinDistance is the minimum distance (in meters) a sample must span
	// before its speed is trusted. Shorter spans are merged into the
	// following sample.
	MinDistance float64

	// MaxHDOP discards points whose horizontal dilution of precision exceeds
	// this value, bridging the gap between the surrounding good fixes. Zero
	// disables HDOP filtering.
	MaxHDOP float64

	// MaxSpeedKmh excludes samples faster than this from the histogram and
	// percentiles, reporting them separately as GPS noise. Zero disables
	// this clamp.
	MaxSpeedKmh float64

	// BinWidth is the histogram bucket width, km/h. Defaults to 5 if zero.
	BinWidth float64
}

// Sample is a speed measurement merged from one or more consecutive points.
type Sample struct {
	SpeedKmh float64
	Distance float64 // meters
	Duration float64 // seconds
}

// Bucket is one histogram bin, covering speeds in [FromKmh, ToKmh).
type Bucket struct {
	FromKmh  float64
	ToKmh    float64
	Count    int
	Distance float64 // meters
	Duration float64 // seconds
}

// Histogram holds bucketed samples plus totals needed to turn buckets and
// percentiles into shares of the overall distance/time.
type Histogram struct {
	BinWidth float64
	Buckets  []Bucket

	// Samples are the non-excluded samples, kept for percentile queries.
	Samples []Sample

	TotalDistance float64 // meters
	TotalDuration float64 // seconds

	// Excluded aggregates samples dropped for exceeding Options.MaxSpeedKmh.
	Excluded Bucket
}

// Samples extracts speed samples from every track segment in gpxFile.
func Samples(gpxFile *gpx.GPX, opt Options) []Sample {
	var samples []Sample

	for _, trk := range gpxFile.Tracks {
		for _, seg := range trk.Segments {
			samples = append(samples, segmentSamples(seg.Points, opt)...)
		}
	}

	return samples
}

func segmentSamples(points []gpx.GPXPoint, opt Options) []Sample {
	var (
		samples  []Sample
		pending  Sample
		havePrev bool
		prev     gpx.GPXPoint
	)

	minDuration := opt.MinInterval.Seconds()

	for i := range points {
		pt := points[i]

		if opt.MaxHDOP > 0 && pt.HorizontalDilution.NotNull() && pt.HorizontalDilution.Value() > opt.MaxHDOP {
			continue // Unreliable fix: skip as a node, bridging prev to the next good point.
		}

		if !havePrev {
			prev = pt
			havePrev = true

			continue
		}

		dt := pt.Timestamp.Sub(prev.Timestamp).Seconds()
		if dt <= 0 {
			continue // Duplicate or out-of-order timestamp: drop this single point.
		}

		pending.Distance += prev.Distance3D(&pt)
		pending.Duration += dt
		prev = pt

		if pending.Duration >= minDuration && pending.Distance >= opt.MinDistance {
			pending.SpeedKmh = speedKmh(pending.Distance, pending.Duration)
			samples = append(samples, pending)
			pending = Sample{}
		}
	}

	if pending.Duration > 0 {
		if len(samples) > 0 {
			last := &samples[len(samples)-1]
			last.Distance += pending.Distance
			last.Duration += pending.Duration
			last.SpeedKmh = speedKmh(last.Distance, last.Duration)
		} else {
			pending.SpeedKmh = speedKmh(pending.Distance, pending.Duration)
			samples = append(samples, pending)
		}
	}

	return samples
}

func speedKmh(distance, duration float64) float64 {
	if duration <= 0 {
		return 0
	}

	return (distance / 1000) / (duration / 60 / 60)
}

// NewHistogram buckets samples by speed, weighting each bucket by both the
// distance and time it covers.
func NewHistogram(samples []Sample, opt Options) Histogram {
	binWidth := opt.BinWidth
	if binWidth <= 0 {
		binWidth = 5
	}

	h := Histogram{BinWidth: binWidth}

	for _, s := range samples {
		if opt.MaxSpeedKmh > 0 && s.SpeedKmh > opt.MaxSpeedKmh {
			h.Excluded.Count++
			h.Excluded.Distance += s.Distance
			h.Excluded.Duration += s.Duration

			continue
		}

		h.Samples = append(h.Samples, s)
		h.TotalDistance += s.Distance
		h.TotalDuration += s.Duration

		idx := int(s.SpeedKmh / binWidth)
		for len(h.Buckets) <= idx {
			from := float64(len(h.Buckets)) * binWidth
			h.Buckets = append(h.Buckets, Bucket{FromKmh: from, ToKmh: from + binWidth})
		}

		b := &h.Buckets[idx]
		b.Count++
		b.Distance += s.Distance
		b.Duration += s.Duration
	}

	return h
}

// WeightFunc selects which sample weight to use for a coverage/percentile
// query: distance-weighted or time-weighted.
type WeightFunc func(Sample) float64

// ByDistance weighs a sample by the distance it covers.
func ByDistance(s Sample) float64 { return s.Distance }

// ByDuration weighs a sample by the time it covers.
func ByDuration(s Sample) float64 { return s.Duration }

// SpeedAbove returns the speed threshold X such that coveragePct percent of
// the total weight (distance or time, per weightFn) was covered at speed at
// or above X. For example SpeedAbove(99, ByDistance) answers "99% of
// distance was covered at speed above X km/h".
func (h Histogram) SpeedAbove(coveragePct float64, weightFn WeightFunc) float64 {
	if len(h.Samples) == 0 {
		return 0
	}

	sorted := make([]Sample, len(h.Samples))
	copy(sorted, h.Samples)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].SpeedKmh < sorted[j].SpeedKmh })

	var total float64
	for _, s := range sorted {
		total += weightFn(s)
	}

	if total == 0 {
		return 0
	}

	target := total * (1 - coveragePct/100)

	var cum float64
	for i, s := range sorted {
		w := weightFn(s)
		if cum+w >= target {
			if w == 0 {
				return s.SpeedKmh
			}

			lo := s.SpeedKmh
			if i > 0 {
				lo = sorted[i-1].SpeedKmh
			}

			frac := (target - cum) / w

			return lo + frac*(s.SpeedKmh-lo)
		}

		cum += w
	}

	return sorted[len(sorted)-1].SpeedKmh
}
