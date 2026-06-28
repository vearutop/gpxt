package sigma_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vearutop/gpxt/sigma"
)

func TestMergeSlfIntoGpx(t *testing.T) {
	require.NoError(t, sigma.MergeSlfIntoGpxFile(
		"testdata/5b7ab817c983b8885c2e1bd64f15dba74ccf7a1b.slf",
		"testdata/RunnerUp_2024-08-10-19-24-37_Biking.gpx",
		"testdata/RunnerUp_2024-08-10-19-24-37_Biking-SLF.gpx",
	))
}

func TestMergeSlfIntoGpx_byDist(t *testing.T) {
	require.NoError(t, sigma.MergeSlfIntoGpxFile(
		"testdata/5b7ab817c983b8885c2e1bd64f15dba74ccf7a1b.slf",
		"testdata/RunnerUp_2024-08-10-19-24-37_Biking.gpx",
		"testdata/RunnerUp_2024-08-10-19-24-37_Biking-SLF.gpx",
		func(options *sigma.MapSlf) {
			options.ByDist = true
		},
	))
}

func TestSlfInfo(t *testing.T) {
	require.NoError(t, sigma.SlfInfo("testdata/2026.06.06.slf"))
	require.NoError(t, sigma.SlfInfo("testdata/2026.06.06-07.slf"))

	sigma.MergeSlfIntoGpxFile("testdata/2026.06.06-07.slf", "testdata/2026.06.07.gpx", "testdata/2026.06.07.slf.gpx", func(options *sigma.MapSlf) {
		options.ByDist = true
		options.SkipStartDist = 83628
	})
}
