package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/spf13/cobra"
)

// pagerDefaultHeader matches the header that real list commands render.
var pagerDefaultHeader = cmdio.Heredoc(`
	{{header "Name"}}	{{header "State"}}`)

// pagerRowTemplate matches the row template used by real list commands.
var pagerRowTemplate = cmdio.Heredoc(`
	{{range .}}{{.Name|green}}	{{.State}}
	{{end}}`)

type pagerRow struct {
	Name  string
	State string
}

// pagerFakeIterator yields synthetic rows. It implements listing.Iterator so
// the production RenderIterator decides between paged and non-paged output.
type pagerFakeIterator struct {
	rows       []pagerRow
	next       int
	failAfter  int // -1 disables the injected error
	delayFirst time.Duration
	delay      time.Duration
}

var _ listing.Iterator[pagerRow] = (*pagerFakeIterator)(nil)

func (it *pagerFakeIterator) HasNext(context.Context) bool {
	if it.failAfter >= 0 && it.next == it.failAfter {
		// Report more items so the caller reaches Next and sees the error.
		return true
	}
	return it.next < len(it.rows)
}

func (it *pagerFakeIterator) Next(ctx context.Context) (pagerRow, error) {
	delay := it.delay
	if it.next == 0 {
		delay = it.delayFirst
	}
	if delay > 0 {
		select {
		case <-ctx.Done():
			return pagerRow{}, ctx.Err()
		case <-time.After(delay):
		}
	}
	if it.failAfter >= 0 && it.next == it.failAfter {
		return pagerRow{}, fmt.Errorf("injected iterator error after %d rows", it.failAfter)
	}
	row := it.rows[it.next]
	it.next++
	return row, nil
}

var pagerStates = []string{"RUNNING", "TERMINATED", "PENDING"}

func newPagerCmd() *cobra.Command {
	var (
		items      int
		header     string
		delayFirst time.Duration
		delay      time.Duration
		failAfter  int
	)
	cmd := &cobra.Command{
		Use:   "pager",
		Short: "cmdio.RenderIterator over synthetic rows (paged only when fully interactive)",
		Args: func(cmd *cobra.Command, args []string) error {
			// cobra runs Args before PersistentPreRunE, which is where cmdio reads
			// these annotations, so the --header flag is applied here.
			cmd.Annotations["headerTemplate"] = cmdio.Heredoc(fmt.Sprintf(`
	{{header %q}}	{{header "State"}}`, header))
			return cobra.NoArgs(cmd, args)
		},
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if items < 0 {
				return fmt.Errorf("--items must be non-negative, got %d", items)
			}
			if failAfter < -1 {
				return fmt.Errorf("--fail-after must be -1 or greater, got %d", failAfter)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			it := &pagerFakeIterator{
				rows:       buildPagerRows(items),
				failAfter:  failAfter,
				delayFirst: delayFirst,
				delay:      delay,
			}
			return cmdio.RenderIterator(cmd.Context(), it)
		},
		Annotations: map[string]string{
			"headerTemplate": pagerDefaultHeader,
			"template":       pagerRowTemplate,
		},
	}
	cmd.Flags().IntVar(&items, "items", 50, "number of rows to yield")
	cmd.Flags().StringVar(&header, "header", "Name", "text for the Name column header")
	cmd.Flags().DurationVar(&delayFirst, "delay-first", 0, "delay before the first row")
	cmd.Flags().DurationVar(&delay, "delay", 0, "delay before each following row")
	cmd.Flags().IntVar(&failAfter, "fail-after", -1, "inject an iterator error after this many rows (0 fails on the first Next, -1 disables)")
	return cmd
}

func buildPagerRows(n int) []pagerRow {
	rows := make([]pagerRow, 0, n)
	for i := range n {
		rows = append(rows, pagerRow{
			Name:  fmt.Sprintf("item-%04d", i+1),
			State: pagerStates[i%len(pagerStates)],
		})
	}
	return rows
}
