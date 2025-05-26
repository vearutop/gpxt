package convert_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/convert"
)

func TestFromLocus(t *testing.T) {
	data, err := os.ReadFile("testdata/locus.gpx")
	require.NoError(t, err)

	g, err := convert.FromLocus(data)
	require.NoError(t, err)

	println(g.Time.String())

	x, err := g.ToXml(gpx.ToXmlParams{
		Indent: true,
	})
	require.NoError(t, err)

	require.NoError(t, os.WriteFile("testdata/locus-cnv.gpx", x, 0o600))
}
