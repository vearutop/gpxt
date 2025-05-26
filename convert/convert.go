package convert

import (
	"github.com/tkrajina/gpxgo/gpx"
)

// Auto tries to load GPX of any known format.
func Auto(data []byte) (gpx.GPX, error) {
	g, err := FromLocus(data)
	if err == nil {
		return g, nil
	}

	return gpx.GPX{}, err
}
