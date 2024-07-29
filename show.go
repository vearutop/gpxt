package main

import (
	"context"
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
	"github.com/vearutop/gpxt/static"
	"github.com/vearutop/statigz"
	"github.com/vearutop/statigz/brotli"
)

func showCmd() {
	var (
		files []string
		tiles string
	)

	info := kingpin.Command("show", "Show GPX file on the map in the browser")
	info.Arg("files", "Files to show on the map.").StringsVar(&files)
	info.Flag("tiles", "URL pattern for map tiles.").
		Default("https://tile.openstreetmap.org/{z}/{x}/{y}.png").
		Envar("LEAFLET_TILES").
		StringVar(&tiles)

	info.Action(func(_ *kingpin.ParseContext) error {
		if len(files) < 1 {
			return errors.New("at least one file expected to show")
		}

		s := web.NewService(openapi3.NewReflector())
		s.Mount("/static/", http.StripPrefix("/static", Static))
		s.Get("/track/{id}.gpx", dlGPX(files))
		s.Get("/", showMap(files, tiles))

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
