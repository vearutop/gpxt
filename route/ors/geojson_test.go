package ors_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vearutop/gpxt/route/ors"
)

func TestGeoJSON_Unmarshal(t *testing.T) {
	var gj ors.GeoJSON

	d, err := os.ReadFile("testdata/ors__v2_directions_{profile}_geojson_post_1699803546246.geojson")
	require.NoError(t, err)

	require.NoError(t, json.Unmarshal(d, &gj))
	fmt.Println(len(gj.Features[0].Geometry.Coordinates))
	_ = gj

	segs := gj.Features[0].Properties.Segments
	points := gj.Features[0].Geometry.Coordinates

	for _, seg := range segs {
		fmt.Printf("%.2fm, %s\n", seg.Distance, time.Duration(seg.Duration*float64(time.Second)).String())
		start := seg.Steps[0]
		end := seg.Steps[len(seg.Steps)-1]
		fmt.Println(points[start.WayPoints[0]], "to", points[end.WayPoints[1]])
		fmt.Println(start.Name, "to", end.Name)
	}
}
