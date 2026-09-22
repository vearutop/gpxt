package main

import (
	"net/http"

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
