package output

import (
	"fmt"
	"io"
	"text/tabwriter"
)

// RenderTable renders a table with headers and rows using tabwriter
func RenderTable(w io.Writer, headers []string, rows [][]string, colors *Colors) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	// Write header
	if len(headers) > 0 {
		for i, h := range headers {
			if i > 0 {
				fmt.Fprint(tw, "\t")
			}
			if colors != nil && colors.Enabled() {
				fmt.Fprint(tw, colors.Bold(h))
			} else {
				fmt.Fprint(tw, h)
			}
		}
		fmt.Fprintln(tw)
	}

	// Write rows
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Fprint(tw, "\t")
			}
			fmt.Fprint(tw, cell)
		}
		fmt.Fprintln(tw)
	}

	return tw.Flush()
}
