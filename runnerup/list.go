package runnerup

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/alecthomas/kingpin/v2"
	"github.com/olekukonko/tablewriter"
)

func list(c *kingpin.CmdClause) {
	var (
		dbFile string
		limit  uint64
	)

	list := c.Command("list", "List latest activities")
	list.Arg("db", "RunnerUp DB export SQLite file.").Required().StringVar(&dbFile)
	list.Flag("limit", "Maximum number of entries in the list.").Default("100").Uint64Var(&limit)

	list.Action(func(_ *kingpin.ParseContext) error {
		r, err := NewRepository(dbFile)
		if err != nil {
			return err
		}

		l, err := r.ListActivities(context.Background(), limit)
		if err != nil {
			return err
		}

		var data [][]string

		for _, i := range l {
			data = append(data, []string{
				strconv.Itoa(i.ID),
				i.StartTime.String(),
				i.Time.String(),
				fmt.Sprintf("%.2f km", i.Distance/1000),
			})
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"ID", "Start Time", "Duration", "Distance"})
		table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
		table.SetCenterSeparator("|")
		table.AppendBulk(data) // Add Bulk Data
		table.Render()

		return nil
	})
}
