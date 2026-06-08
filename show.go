package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"runtime"

	"github.com/alecthomas/kingpin/v2"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest/request"
	"github.com/swaggest/rest/web"
	"github.com/swaggest/usecase"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/static"
	"github.com/vearutop/statigz"
	"github.com/vearutop/statigz/brotli"
)

func showCmd() {
	var (
		files    []string
		tiles    string
		styleURL string
		mapLibre bool
	)

	info := kingpin.Command("show", "Show GPX file on the map in the browser")
	info.Arg("files", "Files to show on the map.").StringsVar(&files)
	info.Flag("tiles", "URL pattern for map tiles.").
		Default("https://tile.openstreetmap.org/{z}/{x}/{y}.png").
		Envar("LEAFLET_TILES").
		StringVar(&tiles)
	info.Flag("style", "MapLibre style URL.").
		Default("https://tiles.openfreemap.org/styles/liberty").
		Envar("MAPLIBRE_STYLE").
		StringVar(&styleURL)
	info.Flag("map-libre", "Show tracks on a MapLibre map instead of Leaflet.").
		Default("false").
		BoolVar(&mapLibre)

	info.Action(func(_ *kingpin.ParseContext) error {
		if len(files) < 1 {
			return errors.New("at least one file expected to show")
		}

		s := web.NewService(openapi3.NewReflector())
		s.Mount("/static/", http.StripPrefix("/static", Static))
		if mapLibre {
			s.Get("/track/{id}.geojson", dlGeoJSON(files))
			s.Get("/", showMapLibre(files, styleURL))
		} else {
			s.Get("/track/{id}.gpx", dlGPX(files))
			s.Get("/", showMap(files, tiles))
		}

		srv := httptest.NewServer(s)

		log.Println("Starting web server at", srv.URL)
		log.Println("Press Ctrl+C to stop")

		if err := openBrowser(srv.URL); err != nil {
			log.Println("open browser:", err.Error())
		}

		<-make(chan struct{})

		return nil
	})
}

// Static serves static assets.
var Static http.Handler

//nolint:gochecknoinits
func init() {
	if _, err := os.Stat("./static"); err == nil {
		// path/to/whatever exists
		Static = http.FileServer(http.Dir("./static"))
	} else {
		Static = statigz.FileServer(static.Assets, brotli.AddEncoding, statigz.EncodeOnInit)
	}
}

// showMap creates use case interactor to show map.
func showMap(files []string, tiles string) usecase.Interactor {
	tmpl, err := static.Template("map.html")
	if err != nil {
		panic(err)
	}

	type pageData struct {
		Files []string
		Tiles string
	}

	u := usecase.NewInteractor(func(_ context.Context, _ struct{}, out *page) error {
		d := pageData{
			Files: files,
			Tiles: tiles,
		}

		return out.Render(tmpl, d)
	})

	return u
}

func showMapLibre(files []string, styleURL string) usecase.Interactor {
	tmpl, err := static.Template("maplibre.html")
	if err != nil {
		panic(err)
	}

	type pageData struct {
		Files []string
		Tiles string
	}

	u := usecase.NewInteractor(func(_ context.Context, _ struct{}, out *page) error {
		d := pageData{
			Files: files,
			Tiles: styleURL,
		}

		return out.Render(tmpl, d)
	})

	return u
}

func dlGPX(files []string) usecase.Interactor {
	type req struct {
		request.EmbeddedSetter

		ID uint `path:"id"`
	}

	u := usecase.NewInteractor(func(_ context.Context, in req, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		if in.ID >= uint(len(files)) {
			return fmt.Errorf("unexpected id %d, max %d", in.ID, len(files))
		}

		http.ServeFile(rw, in.Request(), files[in.ID])

		return nil
	})

	return u
}

func dlGeoJSON(files []string) usecase.Interactor {
	type req struct {
		request.EmbeddedSetter

		ID uint `path:"id"`
	}

	u := usecase.NewInteractor(func(_ context.Context, in req, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		if in.ID >= uint(len(files)) {
			return fmt.Errorf("unexpected id %d, max %d", in.ID, len(files))
		}

		doc, err := gpx.ParseFile(files[in.ID])
		if err != nil {
			return err
		}

		geojson, err := gpxToGeoJSON(doc, files[in.ID])
		if err != nil {
			return err
		}

		rw.Header().Set("Content-Type", "application/geo+json")
		_, err = rw.Write([]byte(geojson))
		return err
	})

	return u
}

func gpxToGeoJSON(doc *gpx.GPX, sourceName string) (string, error) {
	type geometry struct {
		Type        string      `json:"type"`
		Coordinates [][]float64 `json:"coordinates,omitempty"`
	}
	type properties struct {
		Name    string `json:"name,omitempty"`
		Source  string `json:"source,omitempty"`
		Track   string `json:"track,omitempty"`
		Segment int    `json:"segment,omitempty"`
	}
	type feature struct {
		Type       string     `json:"type"`
		Geometry   geometry   `json:"geometry"`
		Properties properties `json:"properties"`
	}
	type featureCollection struct {
		Type     string    `json:"type"`
		Features []feature `json:"features"`
	}

	var features []feature
	for trackIdx, track := range doc.Tracks {
		for segIdx, segment := range track.Segments {
			if len(segment.Points) == 0 {
				continue
			}

			coords := make([][]float64, 0, len(segment.Points))
			for _, pt := range segment.Points {
				coords = append(coords, []float64{pt.Longitude, pt.Latitude})
			}

			name := track.Name
			if name == "" {
				name = fmt.Sprintf("track %d", trackIdx+1)
			}

			features = append(features, feature{
				Type: "Feature",
				Geometry: geometry{
					Type:        "LineString",
					Coordinates: coords,
				},
				Properties: properties{
					Name:    name,
					Source:  sourceName,
					Track:   name,
					Segment: segIdx + 1,
				},
			})
		}
	}

	out, err := json.Marshal(featureCollection{
		Type:     "FeatureCollection",
		Features: features,
	})
	if err != nil {
		return "", err
	}

	return string(out), nil
}

type page struct {
	w io.Writer
}

func (o *page) SetWriter(w io.Writer) {
	o.w = w
}

func (o *page) Render(tmpl *template.Template, data any) error {
	return tmpl.Execute(o.w, data)
}

// openBrowser opens the specified URL in the default browser of the user.
func openBrowser(url string) error {
	var (
		cmd  string
		args []string
	)

	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}

	args = append(args, url)

	return exec.Command(cmd, args...).Start() //nolint:gosec
}
