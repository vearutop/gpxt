package sigma

import (
	"fmt"
	"path"
	"strings"

	"github.com/alecthomas/kingpin/v2"
)

// Cmd defines sigma subcommand.
func Cmd() {
	c := kingpin.Command("sigma", "Operations on Sigma SLF report.")

	merge(c)
}

func merge(c *kingpin.CmdClause) {
	var (
		gpxFile string
		slfFile string
		output  string
		byDist  bool
	)

	list := c.Command("merge", "Merge SLF into GPX")
	list.Arg("gpx", "Source GPX file.").Required().StringVar(&gpxFile)
	list.Arg("slf", "Source SLF file.").Required().StringVar(&slfFile)

	list.Flag("output", "Output file.").Default("<name>.slf.gpx").StringVar(&output)
	list.Flag("by-dist", "Map by distance, can be more accurate than by estimated timestamp.").BoolVar(&byDist)

	list.Action(func(_ *kingpin.ParseContext) error {
		name := strings.TrimSuffix(gpxFile, path.Ext(gpxFile))
		outName := strings.ReplaceAll(output, "<name>", name)

		fmt.Println("Merging " + gpxFile + " with " + slfFile + " into " + outName)

		err := MergeSlfIntoGpx(slfFile, gpxFile, outName, func(options *MapSlf) {
			options.ByDist = byDist
		})
		if err != nil {
			return err
		}

		fmt.Println(outName + " written.")

		return nil
	})
}
