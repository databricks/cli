package cmdio

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

func makeTemplateKeys(bytes ...byte) <-chan byte {
	ch := make(chan byte, len(bytes))
	for _, b := range bytes {
		ch <- b
	}
	close(ch)
	return ch
}

func runPagedTemplate(t *testing.T, n, pageSize int, keys []byte) string {
	t.Helper()
	var out, prompts bytes.Buffer
	iter := listing.Iterator[int](&numberIterator{n: n})
	err := renderIteratorPagedTemplateCore(
		t.Context(),
		iter,
		&out,
		&prompts,
		makeTemplateKeys(keys...),
		"",
		"{{range .}}{{.}}\n{{end}}",
		pageSize,
	)
	require.NoError(t, err)
	return out.String()
}

func TestPagedTemplateBehavior(t *testing.T) {
	tests := []struct {
		name      string
		items     int
		pageSize  int
		keys      []byte
		wantLines int
	}{
		{"drains when first page exhausts iterator", 3, 10, nil, 3},
		{"space fetches one more page", 7, 3, []byte{' '}, 6},
		{"enter drains remaining iterator", 25, 5, []byte{'\r'}, 25},
		{"enter interruptible by ctrl+c", 20, 5, []byte{'\r', pagerKeyCtrlC}, 10},
		{"q exits after first page", 100, 5, []byte{'q'}, 5},
		{"Q exits after first page", 100, 5, []byte{'Q'}, 5},
		{"esc exits after first page", 100, 5, []byte{pagerKeyEscape}, 5},
		{"ctrl+c exits after first page", 100, 5, []byte{pagerKeyCtrlC}, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out := runPagedTemplate(t, tt.items, tt.pageSize, tt.keys)
			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
			assert.Len(t, lines, tt.wantLines)
		})
	}
}

func TestPagedTemplateRespectsLimit(t *testing.T) {
	var out, prompts bytes.Buffer
	iter := listing.Iterator[int](&numberIterator{n: 200})
	ctx := WithLimit(t.Context(), 7)
	err := renderIteratorPagedTemplateCore(
		ctx,
		iter,
		&out,
		&prompts,
		makeTemplateKeys('\r'),
		"",
		"{{range .}}{{.}}\n{{end}}",
		5,
	)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	assert.Len(t, lines, 7)
}

func TestPagedTemplatePrintsHeaderOnce(t *testing.T) {
	var out, prompts bytes.Buffer
	iter := listing.Iterator[int](&numberIterator{n: 8})
	err := renderIteratorPagedTemplateCore(
		t.Context(),
		iter,
		&out,
		&prompts,
		makeTemplateKeys(' '),
		`ID`,
		"{{range .}}{{.}}\n{{end}}",
		3,
	)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(out.String(), "ID\n"))
	assert.True(t, strings.HasPrefix(out.String(), "ID\n"))
}

func TestPagedTemplatePropagatesFetchError(t *testing.T) {
	var out, prompts bytes.Buffer
	iter := listing.Iterator[int](&numberIterator{n: 100, err: errors.New("boom")})
	err := renderIteratorPagedTemplateCore(
		t.Context(),
		iter,
		&out,
		&prompts,
		makeTemplateKeys(),
		"",
		"{{range .}}{{.}}\n{{end}}",
		5,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}

func TestPagedTemplateRendersHeaderAndRowsCorrectly(t *testing.T) {
	var out, prompts bytes.Buffer
	iter := listing.Iterator[int](&numberIterator{n: 6})
	err := renderIteratorPagedTemplateCore(
		t.Context(),
		iter,
		&out,
		&prompts,
		makeTemplateKeys(),
		`ID	Name`,
		"{{range .}}{{.}}	item-{{.}}\n{{end}}",
		100,
	)
	require.NoError(t, err)
	got := out.String()
	assert.Contains(t, got, "ID")
	assert.Contains(t, got, "Name")
	for i := 1; i <= 6; i++ {
		assert.Contains(t, got, fmt.Sprintf("item-%d", i))
	}
	assert.Equal(t, 1, strings.Count(got, "ID"))
}

func TestPagedTemplateEmptyIteratorStillFlushesHeader(t *testing.T) {
	var out, prompts bytes.Buffer
	iter := listing.Iterator[int](&numberIterator{n: 0})
	err := renderIteratorPagedTemplateCore(
		t.Context(),
		iter,
		&out,
		&prompts,
		makeTemplateKeys(),
		`ID	Name`,
		"{{range .}}{{.}}\n{{end}}",
		10,
	)
	require.NoError(t, err)
	assert.Contains(t, out.String(), "ID")
	assert.Contains(t, out.String(), "Name")
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

// TestPagedTemplateMatchesNonPagedForSmallList asserts that single-batch
// output is byte-identical to the non-paged template renderer, so users
// who never hit a second page see the exact same thing they used to.
func TestPagedTemplateMatchesNonPagedForSmallList(t *testing.T) {
	const rows = 5
	tmpl := `{{range .}}{{green "%d" .}}	{{.}}
{{end}}`
	var expected bytes.Buffer
	refIter := listing.Iterator[int](&numberIterator{n: rows})
	require.NoError(t, renderWithTemplate(t.Context(), newIteratorRenderer(refIter), flags.OutputText, &expected, "", tmpl))

	var actual, prompts bytes.Buffer
	pagedIter := listing.Iterator[int](&numberIterator{n: rows})
	require.NoError(t, renderIteratorPagedTemplateCore(
		t.Context(),
		pagedIter,
		&actual,
		&prompts,
		makeTemplateKeys(),
		"",
		tmpl,
		100,
	))
	assert.Equal(t, expected.String(), actual.String())
	assert.NotEmpty(t, expected.String())
}
