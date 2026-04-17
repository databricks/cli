package cmdio

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/databricks/databricks-sdk-go/listing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type pagedJSONRow struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type pagedJSONIterator struct {
	data []pagedJSONRow
	pos  int
	err  error
}

func (it *pagedJSONIterator) HasNext(_ context.Context) bool {
	return it.pos < len(it.data)
}

func (it *pagedJSONIterator) Next(_ context.Context) (pagedJSONRow, error) {
	if it.err != nil {
		return pagedJSONRow{}, it.err
	}
	v := it.data[it.pos]
	it.pos++
	return v, nil
}

func makePagedJSONRows(n int) []pagedJSONRow {
	rows := make([]pagedJSONRow, n)
	for i := range rows {
		rows[i] = pagedJSONRow{ID: i + 1, Name: fmt.Sprintf("row-%d", i+1)}
	}
	return rows
}

func makeKeyChan(keys ...byte) <-chan byte {
	ch := make(chan byte, len(keys))
	for _, k := range keys {
		ch <- k
	}
	close(ch)
	return ch
}

func runPagedJSON(t *testing.T, rows []pagedJSONRow, pageSize int, keys []byte) (string, string, error) {
	t.Helper()
	var out, prompts bytes.Buffer
	iter := listing.Iterator[pagedJSONRow](&pagedJSONIterator{data: rows})
	err := renderIteratorPagedJSONCore(t.Context(), iter, &out, &prompts, makeKeyChan(keys...), pageSize)
	return out.String(), prompts.String(), err
}

func TestPagedJSONRendersFullArrayWhenFitsInOnePage(t *testing.T) {
	got, _, err := runPagedJSON(t, makePagedJSONRows(3), 10, nil)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"id":1,"name":"row-1"},{"id":2,"name":"row-2"},{"id":3,"name":"row-3"}]`, got)
}

func TestPagedJSONSpaceAdvancesOneMorePage(t *testing.T) {
	// 7 items, page=3: first page (3), SPACE → second page (3), then the
	// key channel closes (stdin EOF) and the pager finalizes, writing
	// 6 items total. Fetching item 7 would require a second keypress.
	got, _, err := runPagedJSON(t, makePagedJSONRows(7), 3, []byte{' '})
	require.NoError(t, err)
	var items []pagedJSONRow
	require.NoError(t, json.Unmarshal([]byte(got), &items))
	assert.Len(t, items, 6)
}

func TestPagedJSONTwoSpacesAdvanceTwoMorePages(t *testing.T) {
	// 7 items, page=3, SPACE + SPACE: after the second SPACE the loop
	// fetches item 7, HasNext is false, loop exits and the finalizer
	// closes the array.
	got, _, err := runPagedJSON(t, makePagedJSONRows(7), 3, []byte{' ', ' '})
	require.NoError(t, err)
	var items []pagedJSONRow
	require.NoError(t, json.Unmarshal([]byte(got), &items))
	assert.Len(t, items, 7)
}

func TestPagedJSONEnterDrainsIterator(t *testing.T) {
	got, _, err := runPagedJSON(t, makePagedJSONRows(20), 5, []byte{'\r'})
	require.NoError(t, err)
	var items []pagedJSONRow
	require.NoError(t, json.Unmarshal([]byte(got), &items))
	assert.Len(t, items, 20)
}

func TestPagedJSONQuitEndsEarlyButKeepsValidArray(t *testing.T) {
	got, _, err := runPagedJSON(t, makePagedJSONRows(20), 5, []byte{'q'})
	require.NoError(t, err)
	var items []pagedJSONRow
	require.NoError(t, json.Unmarshal([]byte(got), &items), "output must still be valid JSON after early quit: %q", got)
	assert.Len(t, items, 5, "only the first page should have rendered")
}

func TestPagedJSONEscQuitsWithValidJSON(t *testing.T) {
	got, _, err := runPagedJSON(t, makePagedJSONRows(20), 5, []byte{pagerKeyEscape})
	require.NoError(t, err)
	var items []pagedJSONRow
	require.NoError(t, json.Unmarshal([]byte(got), &items))
	assert.Len(t, items, 5)
}

func TestPagedJSONCtrlCInterruptsDrain(t *testing.T) {
	// ENTER drains, then a buffered Ctrl+C interrupts after the next
	// flush: first page (5) + second page (5) = 10 items; third page
	// skipped due to quit signal.
	got, _, err := runPagedJSON(t, makePagedJSONRows(20), 5, []byte{'\r', pagerKeyCtrlC})
	require.NoError(t, err)
	var items []pagedJSONRow
	require.NoError(t, json.Unmarshal([]byte(got), &items))
	assert.Len(t, items, 10)
}

func TestPagedJSONEmptyIteratorProducesValidEmptyArray(t *testing.T) {
	got, _, err := runPagedJSON(t, nil, 10, nil)
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, got)
}

func TestPagedJSONRespectsLimit(t *testing.T) {
	var out, prompts bytes.Buffer
	iter := listing.Iterator[pagedJSONRow](&pagedJSONIterator{data: makePagedJSONRows(100)})
	ctx := WithLimit(t.Context(), 7)
	err := renderIteratorPagedJSONCore(ctx, iter, &out, &prompts, makeKeyChan('\r'), 5)
	require.NoError(t, err)
	var items []pagedJSONRow
	require.NoError(t, json.Unmarshal(out.Bytes(), &items))
	assert.Len(t, items, 7)
}

func TestPagedJSONFetchErrorPropagatesButStillValidJSON(t *testing.T) {
	var out, prompts bytes.Buffer
	iter := listing.Iterator[pagedJSONRow](&pagedJSONIterator{data: makePagedJSONRows(10), err: errors.New("boom")})
	err := renderIteratorPagedJSONCore(t.Context(), iter, &out, &prompts, makeKeyChan(), 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
	// Partial buffer should still produce valid JSON (empty array).
	assert.JSONEq(t, `[]`, out.String())
}

func TestPagedJSONWritesPromptToPromptsStream(t *testing.T) {
	_, promptsOut, err := runPagedJSON(t, makePagedJSONRows(20), 5, []byte{'q'})
	require.NoError(t, err)
	assert.Contains(t, promptsOut, pagerPromptText)
}
