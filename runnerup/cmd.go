package runnerup

import "github.com/alecthomas/kingpin/v2"

func Cmd() {
	c := kingpin.Command("runnerup", "Operations on RunnerUp DB export file.")

	list(c)
	export(c)
}
