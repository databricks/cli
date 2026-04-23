package cmdio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type numberIterator struct {
	n   int
	pos int
	err error
}

func (it *numberIterator) HasNext(_ context.Context) bool {
	return it.pos < it.n
}

func (it *numberIterator) Next(_ context.Context) (int, error) {
	if it.err != nil {
		return 0, it.err
	}
	it.pos++
	return it.pos, nil
}

// ansiStripPattern is broader than ansiCSIPattern: tea emits non-SGR
// sequences (cursor moves, erase-line, bracketed-paste toggles) that
// the production width calculation doesn't need to strip.
var ansiStripPattern = regexp.MustCompile("\x1b\\[[?]?[0-9;]*[A-Za-z]")

func stripANSI(s string) string {
	return ansiStripPattern.ReplaceAllString(s, "")
}

// pagedOutput runs a full paged render, feeding ENTER to auto-drain,
// and returns the ANSI-stripped output.
func pagedOutput(
	t *testing.T,
	ctx context.Context,
	iter listing.Iterator[int],
	headerTemplate, tmpl string,
	pageSize int,
) string {
	t.Helper()
	var out bytes.Buffer
	require.NoError(t, renderIteratorPagedTemplateCore(
		ctx, iter,
		strings.NewReader("\r"),
		&out,
		headerTemplate, tmpl, pageSize,
	))
	return stripANSI(out.String())
}

func countContentLines(s string) int {
	count := 0
	for line := range strings.SplitSeq(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Contains(trimmed, pagerPromptText) {
			continue
		}
		count++
	}
	return count
}

func TestPagedTemplateDrainsFullIterator(t *testing.T) {
	out := pagedOutput(t, t.Context(), &numberIterator{n: 23}, "", "{{range .}}{{.}}\n{{end}}", 5)
	assert.Equal(t, 23, countContentLines(out))
	for i := 1; i <= 23; i++ {
		assert.Contains(t, out, strconv.Itoa(i))
	}
}

func TestPagedTemplateRespectsLimit(t *testing.T) {
	ctx := WithLimit(t.Context(), 7)
	out := pagedOutput(t, ctx, &numberIterator{n: 200}, "", "{{range .}}{{.}}\n{{end}}", 5)
	assert.Equal(t, 7, countContentLines(out))
}

func TestPagedTemplatePrintsHeaderOnce(t *testing.T) {
	out := pagedOutput(t, t.Context(), &numberIterator{n: 8}, "ID", "{{range .}}{{.}}\n{{end}}", 3)
	assert.Equal(t, 1, strings.Count(out, "ID"))
}

func TestPagedTemplatePropagatesFetchError(t *testing.T) {
	var buf bytes.Buffer
	err := renderIteratorPagedTemplateCore(
		t.Context(),
		&numberIterator{n: 100, err: errors.New("boom")},
		strings.NewReader(""),
		&buf,
		"",
		"{{range .}}{{.}}\n{{end}}",
		5,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestPagedTemplateRendersHeaderAndRows(t *testing.T) {
	out := pagedOutput(t, t.Context(), &numberIterator{n: 6}, "ID\tName", "{{range .}}{{.}}\titem-{{.}}\n{{end}}", 100)
	assert.Contains(t, out, "ID")
	assert.Contains(t, out, "Name")
	for i := 1; i <= 6; i++ {
		assert.Contains(t, out, fmt.Sprintf("item-%d", i))
	}
	assert.Equal(t, 1, strings.Count(out, "ID"))
}

func TestPagedTemplateEmptyIteratorStillFlushesHeader(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	var out bytes.Buffer
	require.NoError(t, renderIteratorPagedTemplateCore(
		t.Context(),
		&numberIterator{n: 0},
		pr,
		&out,
		"ID\tName",
		"{{range .}}{{.}}\n{{end}}",
		10,
	))
	stripped := stripANSI(out.String())
	assert.Contains(t, stripped, "ID")
	assert.Contains(t, stripped, "Name")
}

func TestPagedTemplateColumnsStableAcrossBatches(t *testing.T) {
	it := &numberIterator{n: 6}
	tmpl := "{{range .}}col-{{.}}\tval\n{{end}}"
	out := pagedOutput(t, t.Context(), it, "", tmpl, 3)
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	var dataRows []string
	for _, l := range lines {
		if strings.Contains(l, "col-") {
			dataRows = append(dataRows, l)
		}
	}
	require.Len(t, dataRows, 6)
	// Gap before "val" is the locked column width plus tabwriter minpad.
	for _, row := range dataRows {
		idx := strings.Index(row, "val")
		require.Positive(t, idx)
		assert.GreaterOrEqual(t, idx, len("col-N")+2, "row %q should keep minpad gap", row)
	}
}

// TestPagedTemplateMatchesNonPagedForSmallList pins parity with the
// non-paged path so users who never see a second page see the same
// content they used to.
func TestPagedTemplateMatchesNonPagedForSmallList(t *testing.T) {
	const rows = 5
	tmpl := "{{range .}}{{green \"%d\" .}}\t{{.}}\n{{end}}"

	var expected bytes.Buffer
	refIter := listing.Iterator[int](&numberIterator{n: rows})
	require.NoError(t, renderWithTemplate(t.Context(), newIteratorRenderer(refIter), flags.OutputText, &expected, "", tmpl))

	pagedIter := listing.Iterator[int](&numberIterator{n: rows})
	var actual bytes.Buffer
	pr, pw := io.Pipe()
	defer pw.Close()
	require.NoError(t, renderIteratorPagedTemplateCore(
		t.Context(),
		pagedIter,
		pr,
		&actual,
		"",
		tmpl,
		100,
	))

	assertSameContentLines(t, expected.String(), stripANSI(actual.String()))
}

func assertSameContentLines(t *testing.T, want, got string) {
	t.Helper()
	wantLines := nonEmptyLines(want)
	gotLines := nonEmptyLines(got)
	require.Equal(t, len(wantLines), len(gotLines), "line count mismatch\nwant:\n%s\ngot:\n%s", want, got)
	for i := range wantLines {
		assert.Equal(t, wantLines[i], gotLines[i], "line %d", i)
	}
}

func nonEmptyLines(s string) []string {
	var out []string
	for l := range strings.SplitSeq(s, "\n") {
		t := strings.TrimRight(l, " \r\t")
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}

func TestVisualWidth(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want int
	}{
		{"plain ascii", "hello", 5},
		{"empty", "", 0},
		{"green SGR wraps text", "\x1b[32mhello\x1b[0m", 5},
		{"multiple SGR escapes", "\x1b[1;31mfoo\x1b[0m bar", 7},
		{"multibyte runes count as one each", "héllo", 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, visualWidth(tt.in))
		})
	}
}

func TestComputeWidths(t *testing.T) {
	tests := []struct {
		name string
		rows []string
		want []int
	}{
		{"empty input", nil, nil},
		{"single row", []string{"a\tbb\tccc"}, []int{1, 2, 3}},
		{"widest wins per column", []string{"a\tbb", "aaa\tb"}, []int{3, 2}},
		{"ragged rows extend column count", []string{"a", "b\tcc"}, []int{1, 2}},
		{"SGR escapes don't inflate widths", []string{"\x1b[31mred\x1b[0m\tplain"}, []int{3, 5}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, computeWidths(tt.rows))
		})
	}
}

func TestPadRow(t *testing.T) {
	tests := []struct {
		name   string
		cells  []string
		widths []int
		want   string
	}{
		{"single cell is emitted as-is", []string{"only"}, []int{10}, "only"},
		{"pads every cell except the last", []string{"a", "bb", "c"}, []int{3, 3, 3}, "a    bb   c"},
		{"overflowing cell pushes next column right", []string{"toolong", "b"}, []int{3, 3}, "toolong  b"},
		{"no widths means no padding", []string{"a", "b"}, nil, "a  b"},
		{"SGR escape doesn't count toward pad", []string{"\x1b[31mred\x1b[0m", "b"}, []int{5, 1}, "\x1b[31mred\x1b[0m    b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, padRow(tt.cells, tt.widths))
		})
	}
}
