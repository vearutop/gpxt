package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/swaggest/usecase"
	"github.com/vearutop/gpxt/static"
)

// serveCameraPage serves the self-contained landscape camera preview. It carries no
// server-side state — position, look direction, lens and sun all live in the URL's query
// string and are read/written entirely client-side (see static/camera.html), so reopening
// a shared link reproduces the exact framing someone else set up.
func serveCameraPage() http.HandlerFunc {
	data, err := static.Assets.ReadFile("camera.html")
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(data)
	}
}

// overpassBuildingsURL is a public global mirror that mirrors the whole planet. The main
// overpass-api.de instance flatly blocks browser CORS requests, and (found by testing) at
// least this mirror also blocks direct cross-site *browser* fetches specifically — plain
// server-to-server requests go through fine — hence proxying through our own backend
// instead of querying it from static/camera.html's JS directly.
const overpassBuildingsURL = "https://overpass.openstreetmap.fr/api/interpreter"

// maxBuildingsBBoxDeg caps the query's bounding box, since this endpoint is reachable by
// anyone hitting our public server directly, not just our own JS with its small fixed
// radius — without a cap, it would be an open amplifying proxy onto the upstream service.
const maxBuildingsBBoxDeg = 0.1

// This mirror 403s a request whose User-Agent looks like a browser (it wants exactly this
// — a real backend, not a page hotlinking it client-side) or looks like a bare script
// (Go's own default UA is blocked too); an honest, identifying UA gets through. Found by
// testing directly: default Go client -> 403, a Chrome UA -> 403, this string -> 200.
const overpassUserAgent = "gpxt/1.0 (+https://github.com/vearutop/gpxt)"

var overpassClient = &http.Client{Timeout: 30 * time.Second}

type overpassBuildingsInput struct {
	South float64 `query:"south"`
	West  float64 `query:"west"`
	North float64 `query:"north"`
	East  float64 `query:"east"`
}

func serveOverpassBuildings() usecase.Interactor {
	return usecase.NewInteractor(func(ctx context.Context, in overpassBuildingsInput, out *usecase.OutputWithEmbeddedWriter) error {
		rw, ok := out.Writer.(http.ResponseWriter)
		if !ok {
			return errors.New("missing http.ResponseWriter")
		}

		if in.North <= in.South || in.East <= in.West ||
			in.North-in.South > maxBuildingsBBoxDeg || in.East-in.West > maxBuildingsBBoxDeg {
			return fmt.Errorf("bounding box invalid or too large (max %g° per side)", maxBuildingsBBoxDeg)
		}

		query := fmt.Sprintf(`[out:json][timeout:25];way["building"](%f,%f,%f,%f);out geom;`,
			in.South, in.West, in.North, in.East)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, overpassBuildingsURL,
			strings.NewReader("data="+url.QueryEscape(query)))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("User-Agent", overpassUserAgent)

		resp, err := overpassClient.Do(req)
		if err != nil {
			return fmt.Errorf("querying overpass: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("overpass query failed: HTTP %d", resp.StatusCode)
		}

		rw.Header().Set("Content-Type", "application/json")
		_, err = io.Copy(rw, resp.Body)

		return err
	})
}
