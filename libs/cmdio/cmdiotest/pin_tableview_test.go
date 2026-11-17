package cmdiotest_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/cmdio/cmdiotest/termtest"
	"github.com/databricks/cli/libs/tableview"
)

// TestTableviewPin_SearchInput pins what tableview's search input does with
// Alt+f and a bracketed paste, and the styled screen around it.
func TestTableviewPin_SearchInput(t *testing.T) {
	t.Parallel()
	rows := make([][]string, 0, 20)
	for i := 1; i <= 20; i++ {
		rows = append(rows, []string{fmt.Sprintf("item-%02d", i), "ready"})
	}
	tm := termtest.Run(t, func(ctx context.Context) (struct{}, error) {
		in, out := termtest.Streams(ctx)
		return struct{}{}, tableview.RunWithOptions(ctx, out, []string{"NAME", "STATE"}, rows, tableview.RunOptions{
			Input:  in,
			Width:  cmdio.TestTerminalWidth,
			Height: cmdio.TestTerminalHeight,
		})
	})
	tm.WaitFor("item-01")
	tm.GoldenStyled("01-initial")
	typeKeys(tm, "/", "it", termtest.KeyAltF, termtest.Paste("em-1"))
	tm.Golden("02-search-input")
	tm.Type(termtest.KeyEnter)
	tm.GoldenStyled("03-after-search")
	tm.Type("q")
	_, err := tm.Result()
	termtest.GoldenText(t, "04-result", fmt.Sprintf("err: %v", err))
}
