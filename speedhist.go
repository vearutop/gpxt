package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/olekukonko/tablewriter"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/speedstat"
)

// defaultSpeedPercentiles are the coverage percentages reported for
// "P% of distance/time was covered at speed above X km/h" lines.
var defaultSpeedPercentiles = []float64{50, 75, 90, 95, 99}

// SpeedHistReport renders a speed histogram (bucketed by distance and time)
// plus distance/time-weighted percentile coverage lines for gpxFile.
func SpeedHistReport(gpxFile *gpx.GPX, opt speedstat.Options) string {
	samples := speedstat.Samples(gpxFile, opt)
	h := speedstat.NewHistogram(samples, opt)

	var buf bytes.Buffer

	fmt.Fprintf(&buf, "Speed histogram (bin=%gkm/h, min-interval=%s, min-dist=%gm):\n",
		h.BinWidth, opt.MinInterval, opt.MinDistance)

	table := tablewriter.NewWriter(&buf)
	table.SetHeader([]string{"Speed, km/h", "Count", "Distance", "Time", "Dist %", "Time %"})
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")

	for _, b := range h.Buckets {
		if b.Count == 0 {
			continue
		}

		table.Append([]string{
			fmt.Sprintf("%g-%g", b.FromKmh, b.ToKmh),
			fmt.Sprintf("%d", b.Count),
			fmt.Sprintf("%.2fkm", b.Distance/1000),
			formatDuration(b.Duration),
			fmt.Sprintf("%.1f%%", pct(b.Distance, h.TotalDistance)),
			fmt.Sprintf("%.1f%%", pct(b.Duration, h.TotalDuration)),
		})
	}

	table.Render()

	if h.Excluded.Count > 0 {
		fmt.Fprintf(&buf, "Excluded as GPS noise (>%gkm/h): %d samples, %.2fkm, %s\n",
			opt.MaxSpeedKmh, h.Excluded.Count, h.Excluded.Distance/1000, formatDuration(h.Excluded.Duration))
	}

	buf.WriteString("\nDistance-weighted coverage:\n")

	for _, p := range defaultSpeedPercentiles {
		fmt.Fprintf(&buf, " %g%% of distance was covered at speed above %.1f km/h\n",
			p, h.SpeedAbove(p, speedstat.ByDistance))
	}

	buf.WriteString("\nTime-weighted coverage:\n")

	for _, p := range defaultSpeedPercentiles {
		fmt.Fprintf(&buf, " %g%% of time was covered at speed above %.1f km/h\n",
			p, h.SpeedAbove(p, speedstat.ByDuration))
	}

	return buf.String()
}

func pct(part, total float64) float64 {
	if total == 0 {
		return 0
	}

	return part / total * 100
}

func formatDuration(seconds float64) string {
	return (time.Duration(seconds) * time.Second).String()
}
