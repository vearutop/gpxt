package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/alecthomas/kingpin/v2"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/ns"
)

func csvCmd() {
	var (
		file   string
		output string
	)

	cmd := kingpin.Command("csv", "Export GPX track points to CSV")
	cmd.Arg("file", "GPX file to export.").Required().StringVar(&file)
	cmd.Flag("output", "Output file.").Default("<name>.csv").StringVar(&output)

	cmd.Action(func(_ *kingpin.ParseContext) error {
		gpxFile, err := gpx.ParseFile(file)
		if err != nil {
			return fmt.Errorf("error opening gpx file: %w", err)
		}

		name := strings.TrimSuffix(file, path.Ext(file))
		outName := strings.ReplaceAll(output, "<name>", name)

		f, err := os.Create(outName)
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		defer func() {
			_ = f.Close()
		}()

		w := csv.NewWriter(f)
		if err := w.Write([]string{
			"track",
			"segment",
			"index",
			"time",
			"lat",
			"lon",
			"ele",
			"hr",
			"cad",
			"power",
			"atemp",
		}); err != nil {
			return fmt.Errorf("write header: %w", err)
		}

		for ti, tr := range gpxFile.Tracks {
			for si, s := range tr.Segments {
				for pi, point := range s.Points {
					ele := ""
					if point.Elevation.NotNull() {
						ele = fmt.Sprintf("%.3f", point.Elevation.Value())
					}

					timeVal := ""
					if !point.Timestamp.IsZero() {
						timeVal = point.Timestamp.UTC().Format(time.RFC3339)
					}

					power := extensionDataTPX(&point, "power")
					if power == "" {
						power = extensionData(&point, gpx.NoNamespace, "power")
					}

					if err := w.Write([]string{
						fmt.Sprintf("%d", ti),
						fmt.Sprintf("%d", si),
						fmt.Sprintf("%d", pi),
						timeVal,
						fmt.Sprintf("%.7f", point.Latitude),
						fmt.Sprintf("%.7f", point.Longitude),
						ele,
						extensionDataTPX(&point, "hr"),
						extensionDataTPX(&point, "cad"),
						power,
						extensionDataTPX(&point, "atemp"),
					}); err != nil {
						return fmt.Errorf("write row: %w", err)
					}
				}
			}
		}

		w.Flush()
		if err := w.Error(); err != nil {
			return fmt.Errorf("flush csv: %w", err)
		}

		return nil
	})
}

func extensionData(point *gpx.GPXPoint, namespace gpx.NamespaceURL, path ...string) string {
	node := findExtensionNode(&point.Extensions, namespace, path...)
	if node == nil {
		return ""
	}
	return strings.TrimSpace(node.Data)
}

func extensionDataTPX(point *gpx.GPXPoint, tag string) string {
	if val := extensionData(point, ns.TpxNs, ns.TpxPath, tag); val != "" {
		return val
	}
	if val := extensionData(point, gpx.AnyNamespace, ns.TpxPath, tag); val != "" {
		return val
	}
	return extensionData(point, gpx.NoNamespace, ns.TpxPath, tag)
}

func findExtensionNode(ext *gpx.Extension, namespace gpx.NamespaceURL, path ...string) *gpx.ExtensionNode {
	if len(path) == 0 {
		return nil
	}

	node, ok := ext.GetNode(namespace, path[0])
	if !ok {
		return nil
	}

	for _, part := range path[1:] {
		next, found := node.GetNode(part)
		if !found {
			return nil
		}
		node = next
	}

	return node
}
