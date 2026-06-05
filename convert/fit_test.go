package convert_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vearutop/gpxt/convert"
)

func TestFromFit(t *testing.T) {
	data, err := os.Open("../testdata/20260228.fit")
	require.NoError(t, err)

	g, err := convert.FromFit(data)
	require.NoError(t, err)
	require.Greater(t, g.GetTrackPointsNo(), 0)

	require.NoError(t, data.Close())
}
