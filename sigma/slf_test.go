package sigma_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vearutop/gpxt/sigma"
)

func TestMergeSlfIntoGpx(t *testing.T) {
	require.NoError(t, sigma.MergeSlfIntoGpx(
		"testdata/5b7ab817c983b8885c2e1bd64f15dba74ccf7a1b.slf",
		"testdata/RunnerUp_2024-08-10-19-24-37_Biking.gpx",
		"testdata/RunnerUp_2024-08-10-19-24-37_Biking-SLF.gpx",
	))
}
