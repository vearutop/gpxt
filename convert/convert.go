package convert

import (
	"io"

	"github.com/tkrajina/gpxgo/gpx"
)

// Auto tries to load GPX of any known format.
func Auto(data io.ReadSeeker) (gpx.GPX, error) {
	_, err := data.Seek(0, io.SeekStart)
	if err != nil {
		return gpx.GPX{}, err
	}

	g, err := FromFit(data)
	if err == nil {
		return g, nil
	}

	_, err = data.Seek(0, io.SeekStart)
	if err != nil {
		return gpx.GPX{}, err
	}

	g, err = FromLocus(data)
	if err == nil {
		return g, nil
	}

	return gpx.GPX{}, err
}
