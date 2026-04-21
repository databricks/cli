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

func TestPagedTemplateDrainsWhenFirstPageExhausts(t *testing.T) {
	out := runPagedTemplate(t, 3, 10, nil)
	require.Equal(t, "1\n2\n3\n", out)
}

func TestPagedTemplateSpaceFetchesOneMorePage(t *testing.T) {
	out := runPagedTemplate(t, 7, 3, []byte{' '})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 6)
}

func TestPagedTemplateEnterDrainsIterator(t *testing.T) {
	out := runPagedTemplate(t, 25, 5, []byte{'\r'})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 25)
}

func TestPagedTemplateQuitKeyExits(t *testing.T) {
	out := runPagedTemplate(t, 100, 5, []byte{'q'})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 5)
}

func TestPagedTemplateEscExits(t *testing.T) {
	out := runPagedTemplate(t, 100, 5, []byte{pagerKeyEscape})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 5)
}

func TestPagedTemplateCtrlCExits(t *testing.T) {
	out := runPagedTemplate(t, 100, 5, []byte{pagerKeyCtrlC})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 5)
}

func TestPagedTemplateEnterInterruptibleByCtrlC(t *testing.T) {
	out := runPagedTemplate(t, 20, 5, []byte{'\r', pagerKeyCtrlC})
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	assert.Len(t, lines, 10)
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
	// Regression guard against the header/row template cross-pollution
	// bug: if the two templates share a *template.Template receiver,
	// every flush re-emits the header text where rows should be.
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

// slicedIterator is a tiny iterator implementation for tests that prefer
// to hand over strongly-typed row structs.
type slicedIterator[T any] struct {
	data []T
	pos  int
}

func (it *slicedIterator[T]) HasNext(_ context.Context) bool { return it.pos < len(it.data) }
func (it *slicedIterator[T]) Next(_ context.Context) (T, error) {
	v := it.data[it.pos]
	it.pos++
	return v, nil
}

func TestPagedTemplateColumnWidthsStableAcrossBatches(t *testing.T) {
	type row struct {
		Name string
		Tag  string
	}
	rows := []row{
		{"wide-name-that-sets-the-width", "a"},
		{"short", "b"},
	}
	iter := &slicedIterator[row]{data: rows}
	var out, prompts bytes.Buffer
	err := renderIteratorPagedTemplateCore(
		t.Context(),
		iter,
		&out,
		&prompts,
		makeTemplateKeys(' '),
		"Name	Tag",
		"{{range .}}{{.Name}}	{{.Tag}}\n{{end}}",
		1,
	)
	require.NoError(t, err)
	got := out.String()
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	require.Len(t, lines, 3)

	// Column 1 must start at the same visible offset on every line.
	const wantColTwoOffset = 31
	for i, line := range lines {
		idx := strings.LastIndex(line, " ") + 1
		assert.Equal(t, wantColTwoOffset, idx, "line %d: col 1 expected at offset %d, got %d (line=%q)", i, wantColTwoOffset, idx, line)
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
