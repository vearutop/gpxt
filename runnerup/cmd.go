// Package runnerup provides commands to work with DB exported from RunnerUp app.
package runnerup

import "github.com/alecthomas/kingpin/v2"

// Cmd defines runnerup subcommand.
func Cmd() {
	c := kingpin.Command("runnerup", "Operations on RunnerUp DB export file.")

	list(c)
	export(c)
}
