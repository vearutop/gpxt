package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/usecase"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/static"
)

type gpxStats struct {
	Name           string  `json:"name"`
	DistanceKm     float64 `json:"distance_km"`
	MovingTime     string  `json:"moving_time"`
	MovingTimeSecs float64 `json:"moving_time_secs"`
	AvgSpeedKmh    float64 `json:"avg_speed_kmh"`
	MaxSpeedKmh    float64 `json:"max_speed_kmh"`
	UphillM        float64 `json:"uphill_m"`
	DownhillM      float64 `json:"downhill_m"`
	Date           string  `json:"date"`
	DateStart      string  `json:"date_start"`
}

func posterCmd() {
	var (
		files              []string
		imageFile          string
		styleURL           string
		elevationThreshold float64
	)

	cmd := kingpin.Command("poster", "Create a shareable poster with GPX track on a background image")
	cmd.Arg("files", "GPX files to show on the poster.").StringsVar(&files)
	cmd.Flag("image", "Background image file (JPG/PNG).").Required().StringVar(&imageFile)
	cmd.Flag("style", "MapLibre style URL.").
		Default("https://tiles.openfreemap.org/styles/liberty").
		Envar("MAPLIBRE_STYLE").
		StringVar(&styleURL)
	cmd.Flag("elevation-threshold", "Minimum elevation change (m) counted as gain/loss, filters GPS/barometric noise.").
		Default("8").FloatVar(&elevationThreshold)

	cmd.Action(func(_ *kingpin.ParseContext) error {
		if len(files) < 1 {
			return errors.New("at least one GPX file expected")
		}

		s := web.NewService(openapi3.NewReflector())
		s.Mount("/static/", http.StripPrefix("/static", Static))
		s.Get("/track/{id}.geojson", dlGeoJSON(files))
		s.Get("/image", posterServeImage(imageFile))
		s.Get("/stats.json", posterServeStats(files, elevationThreshold))
		s.Get("/profiles.json", posterServeProfiles(files))

		// If --style points to a local file, serve it from the embedded server
		// so the browser can fetch it from the same origin as the page.
		styleEndpoint := styleURL
		if !strings.HasPrefix(styleURL, "http://") && !strings.HasPrefix(styleURL, "https://") {
			styleData, err := os.ReadFile(styleURL)
			if err != nil {
				return fmt.Errorf("reading style file %s: %w", styleURL, err)
			}
			s.Get("/style.json", posterServeRaw(styleData, "application/json"))
			styleEndpoint = "/style.json"
			log.Println("Serving local style from", styleURL)
		}

		s.Get("/", posterShowPage(files, styleEndpoint))

		srv := httptest.NewServer(s)

		log.Println("Starting poster at", srv.URL)
		log.Println("Press Ctrl+C to stop")

		if err := openBrowser(srv.URL); err != nil {
			log.Println("open browser:", err.Error())
		}

		<-make(chan struct{})

		return nil
	})
}

// posterPageData feeds poster.html. BasePath is empty for the local `poster` command
// (assets served at "/") and a per-session prefix like "/poster/abc123" in server mode.
type posterPageData struct {
	Files    []string
	StyleURL string
	BasePath string
}

func posterShowPage(files []string, styleURL string) usecase.Interactor {
	tmpl, err := static.Template("poster.html")
	if err != nil {
		panic(err)
	}

	return usecase.NewInteractor(func(_ context.Context, _ struct{}, out *page) error {
		return out.Render(tmpl, posterPageData{Files: files, StyleURL: styleURL})
	})
}

func posterServeImage(imageFile string) usecase.Interactor {
	data, err := os.ReadFile(imageFile)
	if err != nil {
		panic(fmt.Sprintf("failed to read image %s: %v", imageFile, err))
	}

	ext := strings.ToLower(filepath.Ext(imageFile))
	ct := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".webp": "image/webp",
	}[ext]
	if ct == "" {
		ct = "image/jpeg"
	}

	return usecase.NewInteractor(func(_ context.Context, _ struct{}, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		rw.Header().Set("Content-Type", ct)
		_, err := rw.Write(data)

		return err
	})
}

func posterServeStats(files []string, elevationThreshold float64) usecase.Interactor {
	return usecase.NewInteractor(func(_ context.Context, _ struct{}, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		allStats, err := computePosterStats(files, elevationThreshold)
		if err != nil {
			return err
		}

		rw.Header().Set("Content-Type", "application/json")

		return json.NewEncoder(rw).Encode(allStats)
	})
}

// computePosterStats is shared by the local `poster` command and `serve`'s per-session
// routes — the only difference between them is where the file list comes from.
func computePosterStats(files []string, elevationThreshold float64) ([]gpxStats, error) {
	var allStats []gpxStats

	for _, f := range files {
		doc, err := gpx.ParseFile(f)
		if err != nil {
			return nil, err
		}

		md := doc.MovingData()
		uphill, downhill := elevationGain(doc, elevationThreshold)
		tb := doc.TimeBounds()

		name := ""
		if len(doc.Tracks) > 0 {
			name = doc.Tracks[0].Name
		}

		if name == "" {
			name = strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		}

		dist := doc.Length3D() / 1000.0
		avgSpd := 0.0

		if md.MovingTime > 0 {
			avgSpd = (md.MovingDistance / md.MovingTime) * 3.6
		}

		date := ""
		if !tb.StartTime.IsZero() {
			date = tb.StartTime.Format("January 2, 2006")
		}

		allStats = append(allStats, gpxStats{
			Name:           name,
			DistanceKm:     dist,
			MovingTime:     posterFormatDuration(time.Duration(md.MovingTime) * time.Second),
			MovingTimeSecs: md.MovingTime,
			AvgSpeedKmh:    avgSpd,
			MaxSpeedKmh:    md.MaxSpeed * 3.6,
			UphillM:        uphill,
			DownhillM:      downhill,
			Date:           date,
			DateStart:      tb.StartTime.Format(time.RFC3339),
		})
	}

	return allStats, nil
}

// elevationGain computes total ascent/descent using a hysteresis threshold instead of
// summing every raw point-to-point delta, which mistakes GPS/barometric jitter for gain.
// Gpxgo's built-in UphillDownhill only applies a light 3-point weighted average, which is
// not enough to filter that noise out (observed ~70% overcount vs Strava on 1m-resolution tracks).
func elevationGain(doc *gpx.GPX, thresholdM float64) (uphill, downhill float64) {
	var (
		base    float64
		hasBase bool
	)

	// Segments within a file are brief recording pauses (stoplights, breaks), not
	// teleports, so the elevation profile is treated as one continuous stream —
	// resetting the baseline at every segment boundary would drop real gain/loss
	// that happened across the pause.
	for _, trk := range doc.Tracks {
		for _, seg := range trk.Segments {
			for _, e := range seg.Elevations() {
				if !e.NotNull() {
					continue
				}

				if !hasBase {
					base, hasBase = e.Value(), true
					continue
				}

				if d := e.Value() - base; d > thresholdM {
					uphill += d
					base = e.Value()
				} else if -d > thresholdM {
					downhill -= d
					base = e.Value()
				}
			}
		}
	}

	return uphill, downhill
}

type metricPoint struct {
	DistKm float64 `json:"d"`
	Value  float64 `json:"v"`
}

// trackProfiles holds every mini-chart series for one file. A metric with no data in the
// source (most files won't have power, for instance) just comes back as a nil slice.
type trackProfiles struct {
	Name      string        `json:"name"`
	Elevation []metricPoint `json:"elevation"`
	HeartRate []metricPoint `json:"hr"`
	Power     []metricPoint `json:"power"`
	Temp      []metricPoint `json:"atemp"`
	Speed     []metricPoint `json:"speed"`
}

// profileBuckets caps each chart's point count regardless of the source track's density
// — a poster-sized chart can't show more detail than this anyway.
const profileBuckets = 200

func posterServeProfiles(files []string) usecase.Interactor {
	return usecase.NewInteractor(func(_ context.Context, _ struct{}, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		profiles, err := computePosterProfiles(files)
		if err != nil {
			return err
		}

		rw.Header().Set("Content-Type", "application/json")

		return json.NewEncoder(rw).Encode(profiles)
	})
}

// computePosterProfiles is shared by the local `poster` command and `serve`'s
// per-session routes — the only difference between them is where the file list comes from.
func computePosterProfiles(files []string) ([]trackProfiles, error) {
	profiles := make([]trackProfiles, 0, len(files))

	for _, f := range files {
		doc, err := gpx.ParseFile(f)
		if err != nil {
			return nil, err
		}

		name := ""
		if len(doc.Tracks) > 0 {
			name = doc.Tracks[0].Name
		}

		if name == "" {
			name = strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		}

		profiles = append(profiles, buildTrackProfiles(doc, name, profileBuckets))
	}

	return profiles, nil
}

// profileSample is one track point's worth of data across every mini-chart metric, with
// per-metric presence flags since most devices don't record all of them.
type profileSample struct {
	dist                                       float64
	ele, hr, power, temp, speed                float64
	hasEle, hasHr, hasPower, hasTemp, hasSpeed bool
}

func buildTrackProfiles(doc *gpx.GPX, name string, buckets int) trackProfiles {
	var (
		samples  []profileSample
		last     gpx.Point
		lastTime time.Time
		has      bool
		cum      float64
	)

	for _, trk := range doc.Tracks {
		for _, seg := range trk.Segments {
			for i := range seg.Points {
				p := &seg.Points[i]

				var stepDist float64
				if has {
					stepDist = last.Distance2D(&p.Point)
					cum += stepDist
				}

				s := profileSample{dist: cum}

				if p.Elevation.NotNull() {
					s.ele, s.hasEle = p.Elevation.Value(), true
				}

				if v, ok := extensionFloatTPX(p, "hr"); ok {
					s.hr, s.hasHr = v, true
				}

				if v, ok := extensionFloatTPX(p, "power"); ok {
					s.power, s.hasPower = v, true
				} else if v, ok := extensionFloat(p, gpx.AnyNamespace, "power"); ok {
					s.power, s.hasPower = v, true
				}

				if v, ok := extensionFloatTPX(p, "atemp"); ok {
					s.temp, s.hasTemp = v, true
				}

				if v, ok := extensionFloatTPX(p, "speed"); ok {
					s.speed, s.hasSpeed = v*3.6, true // m/s -> km/h
				} else if has && !p.Timestamp.IsZero() && !lastTime.IsZero() {
					// Most devices don't record a speed field at all — Strava and
					// everyone else derive it from consecutive position+time, same as
					// gpxt's own moving-average speed elsewhere.
					if dt := p.Timestamp.Sub(lastTime).Seconds(); dt > 0 {
						s.speed, s.hasSpeed = (stepDist/dt)*3.6, true
					}
				}

				samples = append(samples, s)
				last, lastTime, has = p.Point, p.Timestamp, true
			}
		}
	}

	return trackProfiles{
		Name:      name,
		Elevation: bucketProfile(samples, buckets, func(s profileSample) (float64, bool) { return s.ele, s.hasEle }),
		HeartRate: bucketProfile(samples, buckets, func(s profileSample) (float64, bool) { return s.hr, s.hasHr }),
		Power:     bucketProfile(samples, buckets, func(s profileSample) (float64, bool) { return s.power, s.hasPower }),
		Temp:      bucketProfile(samples, buckets, func(s profileSample) (float64, bool) { return s.temp, s.hasTemp }),
		Speed:     bucketProfile(samples, buckets, func(s profileSample) (float64, bool) { return s.speed, s.hasSpeed }),
	}
}

// bucketProfile reduces samples to a fixed number of evenly-spaced-by-distance buckets,
// averaging each metric's value within one. This is a purely visual smoothing for chart
// display — unlike elevationGain's hysteresis threshold, it doesn't need to agree with
// any printed number, just look like a plausible profile at poster size.
func bucketProfile(samples []profileSample, buckets int, get func(profileSample) (float64, bool)) []metricPoint {
	if len(samples) == 0 {
		return nil
	}

	total := samples[len(samples)-1].dist
	if total == 0 {
		if v, ok := get(samples[0]); ok {
			return []metricPoint{{DistKm: 0, Value: v}}
		}

		return nil
	}

	bucketW := total / float64(buckets)
	sums := make([]float64, buckets+1)
	counts := make([]int, buckets+1)

	for _, s := range samples {
		v, ok := get(s)
		if !ok {
			continue
		}

		bi := min(int(s.dist/bucketW), buckets)
		sums[bi] += v
		counts[bi]++
	}

	var points []metricPoint

	for bi, n := range counts {
		if n == 0 {
			continue
		}

		points = append(points, metricPoint{
			DistKm: float64(bi) * bucketW / 1000,
			Value:  sums[bi] / float64(n),
		})
	}

	return points
}

func posterServeRaw(data []byte, contentType string) usecase.Interactor {
	return usecase.NewInteractor(func(_ context.Context, _ struct{}, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		rw.Header().Set("Content-Type", contentType)
		_, err := rw.Write(data)

		return err
	})
}

func posterFormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh %02dm", h, m)
	}

	return fmt.Sprintf("%dm", int(d.Minutes()))
}
