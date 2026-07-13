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
	"github.com/vearutop/gpxt/ns"
)

// MapSlf defines mapping options.
type MapSlf struct {
	ByDist          bool
	SkipStartDist   float64
	KeepStoppedTime bool
	IdleSpeedKmh    float64
}

// defaultIdleSpeedKmh matches gpxgo's defaultStoppedSpeedThreshold, the speed below
// which a point-to-point arc counts as "Stopped time" rather than moving. Used when
// MapSlf.IdleSpeedKmh is left unset (zero or negative).
const defaultIdleSpeedKmh = 1.0

// removeStoppedPoints drops the interior points of stationary dwells (runs of two or
// more consecutive arcs at or below idleSpeedKmh), so parked/jittery clusters don't skew
// distance-based alignment with the SLF stream.
//
// Only interior points are dropped: both endpoints of a dwell are kept, so gpxgo's own
// Moving/Stopped time classification of the resulting track stays close to the original
// - it recomputes speed between whatever points remain, and collapsing a dwell down to
// its two boundary points still leaves a near-zero-distance arc between them, which is
// still classified as stopped. Dropping every stopped point outright (including isolated,
// single-arc ones) instead bridges straight across the dwell to the next point and can
// flip that arc's average speed above the threshold, misclassifying stopped time as moving.
func removeStoppedPoints(gpxFile *gpx.GPX, idleSpeedKmh float64) {
	for ti, tr := range gpxFile.Tracks {
		for si, s := range tr.Segments {
			s.Points = filterStoppedPoints(s.Points, idleSpeedKmh)
			tr.Segments[si] = s
		}

		gpxFile.Tracks[ti] = tr
	}
}

func filterStoppedPoints(points []gpx.GPXPoint, idleSpeedKmh float64) []gpx.GPXPoint {
	if len(points) < 3 {
		return points
	}

	stopped := make([]bool, len(points))

	for i := 1; i < len(points); i++ {
		dt := points[i].Timestamp.Sub(points[i-1].Timestamp).Seconds()
		if dt <= 0 {
			continue
		}

		speedKmh := (points[i-1].Distance3D(&points[i]) / 1000) / (dt / 60 / 60)
		stopped[i] = speedKmh <= idleSpeedKmh
	}

	filtered := make([]gpx.GPXPoint, 0, len(points))
	filtered = append(filtered, points[0])

	for i := 1; i < len(points); {
		if !stopped[i] {
			filtered = append(filtered, points[i])
			i++

			continue
		}

		// Run of one or more stopped arcs: keep only its trailing boundary point,
		// dropping any interior points (points[i-1] is already kept).
		j := i
		for j < len(points) && stopped[j] {
			j++
		}

		filtered = append(filtered, points[j-1])
		i = j
	}

	return filtered
}

// SlfInfo shows information about SLF file.
func SlfInfo(slfFn string) error {
	d, err := os.ReadFile(slfFn) //nolint:gosec
	if err != nil {
		return fmt.Errorf("read source slf: %w", err)
	}

	var v Activity

	if err := xml.Unmarshal(d, &v); err != nil {
		return fmt.Errorf("decode slf: %w", err)
	}

	// Thu Aug 1 17:56:21 GMT+0200 2024
	slfStartTime, err := time.Parse("Mon Jan _2 15:04:05 GMT-0700 2006", v.GeneralInformation.StartDate)
	if err != nil {
		return err
	}

	fmt.Println("Start time:", slfStartTime)

	slfEntries := v.Entries.Entry

	totalSLFDist := slfEntries[len(slfEntries)-1].DistanceAbsolute
	fmt.Printf("SLF dist: %.fm\n", totalSLFDist)

	return nil
}

// MergeSlfIntoGpxFile adds data from SLF into GPX file.
func MergeSlfIntoGpxFile(slfFn, gpxFn, outFn string, opts ...func(options *MapSlf)) error {
	gpxFile, err := gpx.ParseFile(gpxFn)
	if err != nil {
		return fmt.Errorf("parse source gpx: %w", err)
	}

	return MergeSlfIntoGpx(*gpxFile, slfFn, outFn, opts...)
}

// MergeSlfIntoGpx adds data from SLF into GPX file.
func MergeSlfIntoGpx(gpxFile gpx.GPX, slfFn string, outFn string, opts ...func(options *MapSlf)) error {
	var v Activity

	mo := MapSlf{}

	for _, opt := range opts {
		opt(&mo)
	}

	if !mo.KeepStoppedTime {
		idleSpeedKmh := mo.IdleSpeedKmh
		if idleSpeedKmh <= 0 {
			idleSpeedKmh = defaultIdleSpeedKmh
		}

		removeStoppedPoints(&gpxFile, idleSpeedKmh)
	}

	d, err := os.ReadFile(slfFn) //nolint:gosec
	if err != nil {
		return fmt.Errorf("read source slf: %w", err)
	}

	if err := xml.Unmarshal(d, &v); err != nil {
		return fmt.Errorf("decode slf: %w", err)
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
		dist       float64
		prevPoint  *gpx.GPXPoint
		slfEntries = v.Entries.Entry
	)

	totalGPXDist := gpxFile.Length3D()
	fmt.Printf("GPX dist: %.2fkm\n", totalGPXDist/1000.0)

	totalSLFDist := slfEntries[len(slfEntries)-1].DistanceAbsolute
	totalSLFDist -= mo.SkipStartDist
	fmt.Printf("SLF dist: %.2fkm\n", totalSLFDist/1000.0)

	prevPoint = nil

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

	totalGPXDist = dist
	fmt.Printf("GPX dist 2: %.2fkm\n", totalGPXDist/1000.0)

	distRatio := totalSLFDist / totalGPXDist

	fmt.Printf("Dist ratio: %.f%%\n", 100.0*distRatio)

	dist = 0
	prevPoint = nil

	findPowerNode := func(point *gpx.GPXPoint) *gpx.ExtensionNode {
		if tpxNode, found := point.Extensions.GetNode(ns.TpxNs, ns.TpxPath); found {
			if powerNode, ok := tpxNode.GetNode("power"); ok {
				return powerNode
			}
		}

		if powerNode, found := point.Extensions.GetNode(gpx.AnyNamespace, "power"); found {
			return powerNode
		}

		return nil
	}

	visitPoint := func(point *gpx.GPXPoint) {
		if prevPoint != nil {
			dist += distRatio * prevPoint.Distance2D(point)
		}

		prevPoint = point

		// Find closes point by time or by distance.
		i, _ := slices.BinarySearchFunc(v.Entries.Entry, point, func(entry Entry, point *gpx.GPXPoint) int {
			if mo.ByDist {
				if entry.DistanceAbsolute-mo.SkipStartDist < dist {
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

			if mo.ByDist && math.Abs((vv.DistanceAbsolute-mo.SkipStartDist)-dist) > 100.0 {
				return
			}

			if vv.Power != nil {
				node := findPowerNode(point)
				// Power is not present in the original GPX.
				if node == nil {
					node = point.Extensions.GetOrCreateNode(gpx.NoNamespace, "power")
					node.Data = strconv.Itoa(int(*vv.Power))
				}
			}

			if vv.Heartrate != nil && *vv.Heartrate != 0 {
				node := point.Extensions.GetOrCreateNode(ns.TpxNs, ns.TpxPath, "hr")
				if node.Data == "" {
					node.Data = strconv.Itoa(int(*vv.Heartrate))
				}
			}

			if vv.Cadence != nil {
				node := point.Extensions.GetOrCreateNode(ns.TpxNs, ns.TpxPath, "cad")
				if node.Data == "" {
					node.Data = strconv.Itoa(int(*vv.Cadence))
				}
			}

			if vv.Temperature != "" {
				node := point.Extensions.GetOrCreateNode(ns.TpxNs, ns.TpxPath, "atemp")
				if node.Data == "" {
					node.Data = vv.Temperature
				}
			}
		}
	}

	for _, tr := range gpxFile.Tracks {
		for _, s := range tr.Segments {
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
