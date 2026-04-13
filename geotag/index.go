// Package geotag provides lightweight time-based geotag lookup over GPS tracks.
package geotag

import (
	"math"
	"sort"
	"time"

	"github.com/tkrajina/gpxgo/gpx"
)

// Point is a compact GPS fix.
// Altitude is optional; use NaN to mark missing altitude.
type Point struct {
	Time int64   // unix time in nanoseconds
	Lat  float32 // degrees
	Lon  float32 // degrees
	Alt  float32 // meters, NaN if unknown
}

// Options configure index building and lookup behavior.
type Options struct {
	// MergeWithin is the maximum time delta to merge near-duplicate points within a track.
	MergeWithin time.Duration
	// MergeDistMeters is the maximum distance to merge near-duplicate points within a track.
	MergeDistMeters float64

	// MaxGap is the maximum allowed time gap to the nearest fix when geotagging.
	MaxGap time.Duration
	// Interpolate enables linear interpolation between adjacent fixes in time when possible.
	Interpolate bool
}

// Index is a merged, time-ordered slice of points for fast geotag lookup.
type Index struct {
	points []Point
	tracks [][]Point
	loose  []Point
	opts   Options
	dirty  bool

	clocks map[string][]ClockSync
}

// NewIndex creates an empty index with options.
func NewIndex(opts Options) *Index {
	return &Index{opts: opts, dirty: true}
}

// AddTrack appends a time-ordered track to the index builder.
func (idx *Index) AddTrack(points []Point) {
	if len(points) == 0 {
		return
	}
	idx.tracks = append(idx.tracks, points)
	idx.dirty = true
}

// AddPoint appends a single point to the index builder.
// Points added this way are treated as a single loose track for merge purposes.
func (idx *Index) AddPoint(p Point) {
	idx.loose = append(idx.loose, p)
	idx.dirty = true
}

// AddGPX appends tracks from a GPX document.
// Each GPX track segment is treated as a separate track and must be time-ordered.
func (idx *Index) AddGPX(g *gpx.GPX) {
	if g == nil {
		return
	}

	for _, t := range g.Tracks {
		for _, s := range t.Segments {
			if len(s.Points) == 0 {
				continue
			}
			tr := make([]Point, 0, len(s.Points))
			for _, pt := range s.Points {
				if pt.Timestamp.IsZero() {
					continue
				}
				alt := AltUnknown()
				if pt.Elevation.NotNull() {
					alt = float32(pt.Elevation.Value())
				}
				tr = append(tr, Point{
					Time: pt.Timestamp.UnixNano(),
					Lat:  float32(pt.Latitude),
					Lon:  float32(pt.Longitude),
					Alt:  alt,
				})
			}
			if len(tr) > 0 {
				idx.tracks = append(idx.tracks, tr)
			}
		}
	}

	idx.dirty = true
}

// ClockSync defines a time shift for a camera clock within a time range.
// Start or End may be zero to indicate an open-ended range.
type ClockSync struct {
	Start  int64         // unix nanos, inclusive. 0 means -inf.
	End    int64         // unix nanos, inclusive. 0 means +inf.
	Offset time.Duration // added to the photo time to align with GPS time.
}

// AddClockSync registers a time shift for a given clock ID.
// If clockID is empty, "default" is used.
func (idx *Index) AddClockSync(clockID string, start, end time.Time, offset time.Duration) {
	if clockID == "" {
		clockID = "default"
	}
	if idx.clocks == nil {
		idx.clocks = make(map[string][]ClockSync)
	}
	var s, e int64
	if !start.IsZero() {
		s = Time(start)
	}
	if !end.IsZero() {
		e = Time(end)
	}
	cs := ClockSync{Start: s, End: e, Offset: offset}
	idx.clocks[clockID] = append(idx.clocks[clockID], cs)
}

// Build merges, sorts, and finalizes the index.
func (idx *Index) Build() {
	pts := make([]Point, 0)
	for _, tr := range idx.tracks {
		if len(tr) == 0 {
			continue
		}
		merged := mergeTrack(tr, idx.opts.MergeWithin, idx.opts.MergeDistMeters)
		pts = append(pts, merged...)
	}
	if len(idx.loose) > 0 {
		merged := mergeTrack(idx.loose, idx.opts.MergeWithin, idx.opts.MergeDistMeters)
		pts = append(pts, merged...)
	}

	sort.Slice(pts, func(i, j int) bool {
		return pts[i].Time < pts[j].Time
	})

	idx.points = pts
	idx.dirty = false
	// Drop source tracks to reduce memory once index is built.
	idx.tracks = nil
	idx.loose = nil
}

// Points returns the merged, sorted points slice.
func (idx *Index) Points() []Point {
	if idx.dirty {
		idx.Build()
	}
	return idx.points
}

// Lookup finds the closest position to the given time.
// It returns ok=false if the closest fix is farther than MaxGap.
// If Interpolate is enabled and both surrounding fixes are within MaxGap, it interpolates.
func (idx *Index) Lookup(t time.Time) (p Point, ok bool) {
	if idx.dirty {
		idx.Build()
	}
	if len(idx.points) == 0 {
		return Point{}, false
	}

	tn := t.UnixNano()
	i := sort.Search(len(idx.points), func(i int) bool {
		return idx.points[i].Time >= tn
	})

	// Only after.
	if i == 0 {
		gap := time.Duration(idx.points[0].Time - tn)
		if idx.opts.MaxGap > 0 && gap > idx.opts.MaxGap {
			return Point{}, false
		}
		return idx.points[0], true
	}

	// Only before.
	if i >= len(idx.points) {
		gap := time.Duration(tn - idx.points[len(idx.points)-1].Time)
		if idx.opts.MaxGap > 0 && gap > idx.opts.MaxGap {
			return Point{}, false
		}
		return idx.points[len(idx.points)-1], true
	}

	before := idx.points[i-1]
	after := idx.points[i]

	if before.Time == after.Time {
		return after, true
	}

	gapBefore := time.Duration(tn - before.Time)
	gapAfter := time.Duration(after.Time - tn)

	// If interpolation is allowed and both sides are within MaxGap, interpolate.
	if idx.opts.Interpolate {
		if withinMaxGap(idx.opts.MaxGap, gapBefore) && withinMaxGap(idx.opts.MaxGap, gapAfter) {
			return interpolate(before, after, tn), true
		}
	}

	// Otherwise, use nearest fix if within MaxGap.
	if gapBefore <= gapAfter {
		if withinMaxGap(idx.opts.MaxGap, gapBefore) {
			return before, true
		}
		return Point{}, false
	}

	if withinMaxGap(idx.opts.MaxGap, gapAfter) {
		return after, true
	}

	return Point{}, false
}

// LookupWithClock applies clock synchronization and looks up the closest position.
// If no clock sync matches, it falls back to "default", then no offset.
func (idx *Index) LookupWithClock(t time.Time, clockID string) (p Point, ok bool) {
	if clockID == "" {
		clockID = "default"
	}
	offset := idx.clockOffset(clockID, t)
	if offset == 0 && clockID != "default" {
		offset = idx.clockOffset("default", t)
	}
	if offset != 0 {
		t = t.Add(offset)
	}
	return idx.Lookup(t)
}

// OffsetForLocation finds the nearest track point to the given location and
// returns the time offset that would align cameraTime to that fix.
// If maxDistMeters > 0, ok=false is returned when no fix is within that distance.
func (idx *Index) OffsetForLocation(cameraTime time.Time, lat, lon float64, maxDistMeters float64) (offset time.Duration, ok bool) {
	if idx.dirty {
		idx.Build()
	}
	if len(idx.points) == 0 {
		return 0, false
	}

	bestIdx := -1
	bestDist := math.MaxFloat64
	ref := Point{Lat: float32(lat), Lon: float32(lon)}

	for i := range idx.points {
		d := haversineMeters(ref, idx.points[i])
		if d < bestDist {
			bestDist = d
			bestIdx = i
		}
	}

	if bestIdx < 0 {
		return 0, false
	}
	if maxDistMeters > 0 && bestDist > maxDistMeters {
		return 0, false
	}

	trackTime := time.Unix(0, idx.points[bestIdx].Time)
	return trackTime.Sub(cameraTime), true
}

func (idx *Index) clockOffset(clockID string, t time.Time) time.Duration {
	if idx.clocks == nil {
		return 0
	}
	list := idx.clocks[clockID]
	if len(list) == 0 {
		return 0
	}
	tn := t.UnixNano()
	for i := len(list) - 1; i >= 0; i-- {
		cs := list[i]
		if inRange(cs, tn) {
			return cs.Offset
		}
	}
	return 0
}

func inRange(cs ClockSync, tn int64) bool {
	if cs.Start != 0 && tn < cs.Start {
		return false
	}
	if cs.End != 0 && tn > cs.End {
		return false
	}
	return true
}

func withinMaxGap(maxGap, gap time.Duration) bool {
	if maxGap <= 0 {
		return true
	}
	return gap <= maxGap
}

func interpolate(a, b Point, tn int64) Point {
	fa := float64(tn-a.Time) / float64(b.Time-a.Time)
	lat := float32(float64(a.Lat) + (float64(b.Lat)-float64(a.Lat))*fa)
	lon := float32(float64(a.Lon) + (float64(b.Lon)-float64(a.Lon))*fa)
	alt := interpAlt(a.Alt, b.Alt, fa)
	return Point{Time: tn, Lat: lat, Lon: lon, Alt: alt}
}

func interpAlt(a, b float32, f float64) float32 {
	av := altValid(a)
	bv := altValid(b)

	switch {
	case av && bv:
		return float32(float64(a) + (float64(b)-float64(a))*f)
	case av:
		return a
	case bv:
		return b
	default:
		return float32(math.NaN())
	}
}

func altValid(a float32) bool {
	return !math.IsNaN(float64(a))
}

func mergeTrack(points []Point, dt time.Duration, dMeters float64) []Point {
	if len(points) == 0 {
		return nil
	}

	out := make([]Point, 0, len(points))
	prev := points[0]
	out = append(out, prev)

	for i := 1; i < len(points); i++ {
		p := points[i]
		if dt > 0 {
			if time.Duration(p.Time-prev.Time) <= dt {
				if dMeters <= 0 || haversineMeters(prev, p) <= dMeters {
					// Keep the most recent point to preserve monotonic time.
					out[len(out)-1] = p
					prev = p
					continue
				}
			}
		}

		out = append(out, p)
		prev = p
	}

	return out
}

const earthRadiusMeters = 6371000.0

func haversineMeters(a, b Point) float64 {
	lat1 := float64(a.Lat) * (math.Pi / 180.0)
	lat2 := float64(b.Lat) * (math.Pi / 180.0)
	lon1 := float64(a.Lon) * (math.Pi / 180.0)
	lon2 := float64(b.Lon) * (math.Pi / 180.0)

	dlat := lat2 - lat1
	dlon := lon2 - lon1

	sinDLat := math.Sin(dlat / 2.0)
	sinDLon := math.Sin(dlon / 2.0)

	a2 := sinDLat*sinDLat + math.Cos(lat1)*math.Cos(lat2)*sinDLon*sinDLon
	c := 2 * math.Atan2(math.Sqrt(a2), math.Sqrt(1-a2))
	return earthRadiusMeters * c
}

// Time converts t to unix nanoseconds.
func Time(t time.Time) int64 {
	return t.UnixNano()
}

// AltUnknown returns a NaN value to mark missing altitude.
func AltUnknown() float32 {
	return float32(math.NaN())
}

// AltKnown reports whether altitude is present.
func AltKnown(a float32) bool {
	return altValid(a)
}
