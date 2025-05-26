package main

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/tkrajina/gpxgo/gpx"
	"github.com/vearutop/gpxt/convert"
)

func convertCmd() {
	var (
		files  []string
		output string
		indent bool
	)

	cmd := kingpin.Command("convert", "Convert files from exotic formats (supported: Locus)")
	cmd.Arg("files", "GPX files to process.").Required().StringsVar(&files)
	cmd.Flag("output", "Output file.").Default("<name>.converted.gpx").StringVar(&output)
	cmd.Flag("indent", "Indent output file.").BoolVar(&indent)

	cmd.Action(func(_ *kingpin.ParseContext) error {
		for _, file := range files {
			data, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("error opening file: %w", err)
			}

			gpxFile, err := convert.Auto(data)
			if err != nil {
				return fmt.Errorf("error converting file: %w", err)
			}

			xx, err := gpxFile.ToXml(gpx.ToXmlParams{Indent: indent})
			if err != nil {
				return fmt.Errorf("render GPX: %w", err)
			}

			name := strings.TrimSuffix(file, path.Ext(file))
			outName := strings.ReplaceAll(output, "<name>", name)

			if err = os.WriteFile(outName, xx, 0o600); err != nil {
				return fmt.Errorf("save GPX file: %w", err)
			}
		}

		return nil
	})
}
