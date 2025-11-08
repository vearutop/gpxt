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
		byTime  bool
	)

	merge := c.Command("merge", "Merge SLF into GPX")
	merge.Arg("gpx", "Source GPX file.").Required().StringVar(&gpxFile)
	merge.Arg("slf", "Source SLF file.").Required().StringVar(&slfFile)

	merge.Flag("output", "Output file.").Default("<name>.slf.gpx").StringVar(&output)
	merge.Flag("by-time", "Map by estimated time, can be less accurate than by distance.").BoolVar(&byTime)

	merge.Action(func(_ *kingpin.ParseContext) error {
		name := strings.TrimSuffix(gpxFile, path.Ext(gpxFile))
		outName := strings.ReplaceAll(output, "<name>", name)

		fmt.Println("Merging " + gpxFile + " with " + slfFile + " into " + outName)

		err := MergeSlfIntoGpx(slfFile, gpxFile, outName, func(options *MapSlf) {
			options.ByDist = !byTime
		})
		if err != nil {
			return err
		}

		fmt.Println(outName + " written.")

		return nil
	})
}
