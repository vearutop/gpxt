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
}

func posterCmd() {
	var (
		files     []string
		imageFile string
		styleURL  string
	)

	cmd := kingpin.Command("poster", "Create a shareable poster with GPX track on a background image")
	cmd.Arg("files", "GPX files to show on the poster.").StringsVar(&files)
	cmd.Flag("image", "Background image file (JPG/PNG).").Required().StringVar(&imageFile)
	cmd.Flag("style", "MapLibre style URL.").
		Default("https://tiles.openfreemap.org/styles/fiord").
		Envar("MAPLIBRE_STYLE").
		StringVar(&styleURL)

	cmd.Action(func(_ *kingpin.ParseContext) error {
		if len(files) < 1 {
			return errors.New("at least one GPX file expected")
		}

		s := web.NewService(openapi3.NewReflector())
		s.Mount("/static/", http.StripPrefix("/static", Static))
		s.Get("/track/{id}.geojson", dlGeoJSON(files))
		s.Get("/image", posterServeImage(imageFile))
		s.Get("/stats.json", posterServeStats(files))

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

func posterShowPage(files []string, styleURL string) usecase.Interactor {
	tmpl, err := static.Template("poster.html")
	if err != nil {
		panic(err)
	}

	type pageData struct {
		Files    []string
		StyleURL string
	}

	return usecase.NewInteractor(func(_ context.Context, _ struct{}, out *page) error {
		return out.Render(tmpl, pageData{Files: files, StyleURL: styleURL})
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

func posterServeStats(files []string) usecase.Interactor {
	return usecase.NewInteractor(func(_ context.Context, _ struct{}, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		var allStats []gpxStats

		for _, f := range files {
			doc, err := gpx.ParseFile(f)
			if err != nil {
				return err
			}

			md := doc.MovingData()
			updo := doc.UphillDownhill()
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
				UphillM:        updo.Uphill,
				DownhillM:      updo.Downhill,
				Date:           date,
			})
		}

		rw.Header().Set("Content-Type", "application/json")

		return json.NewEncoder(rw).Encode(allStats)
	})
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
