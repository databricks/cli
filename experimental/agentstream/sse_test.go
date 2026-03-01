package agentstream

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSSEReader_SingleEvent(t *testing.T) {
	input := "data: {\"type\":\"response.output_item.added\"}\n\n"
	r := NewSSEReader(strings.NewReader(input))

	ev, err := r.Next()
	require.NoError(t, err)
	assert.Equal(t, `{"type":"response.output_item.added"}`, ev.Data)

	_, err = r.Next()
	assert.ErrorIs(t, err, io.EOF)
}

func TestSSEReader_MultipleEvents(t *testing.T) {
	input := "data: first\n\ndata: second\n\n"
	r := NewSSEReader(strings.NewReader(input))

	ev1, err := r.Next()
	require.NoError(t, err)
	assert.Equal(t, "first", ev1.Data)

	ev2, err := r.Next()
	require.NoError(t, err)
	assert.Equal(t, "second", ev2.Data)

	_, err = r.Next()
	assert.ErrorIs(t, err, io.EOF)
}

func TestSSEReader_EmptyStream(t *testing.T) {
	r := NewSSEReader(strings.NewReader(""))
	_, err := r.Next()
	assert.ErrorIs(t, err, io.EOF)
}

func TestSSEReader_CommentsAndOtherFieldsIgnored(t *testing.T) {
	input := ": this is a comment\nevent: update\ndata: hello\nid: 123\n\n"
	r := NewSSEReader(strings.NewReader(input))

	ev, err := r.Next()
	require.NoError(t, err)
	assert.Equal(t, "hello", ev.Data)
}

func TestSSEReader_DataWithoutSpace(t *testing.T) {
	input := "data:nospace\n\n"
	r := NewSSEReader(strings.NewReader(input))

	ev, err := r.Next()
	require.NoError(t, err)
	assert.Equal(t, "nospace", ev.Data)
}

func TestSSEReader_MultiLineData(t *testing.T) {
	input := "data: line1\ndata: line2\n\n"
	r := NewSSEReader(strings.NewReader(input))

	ev, err := r.Next()
	require.NoError(t, err)
	assert.Equal(t, "line1\nline2", ev.Data)
}

func TestSSEReader_EOFWithoutTrailingBlankLine(t *testing.T) {
	input := "data: trailing"
	r := NewSSEReader(strings.NewReader(input))

	ev, err := r.Next()
	require.NoError(t, err)
	assert.Equal(t, "trailing", ev.Data)

	_, err = r.Next()
	assert.ErrorIs(t, err, io.EOF)
}

func TestSSEReader_BlankLinesWithoutData(t *testing.T) {
	input := "\n\n\ndata: after blanks\n\n"
	r := NewSSEReader(strings.NewReader(input))

	ev, err := r.Next()
	require.NoError(t, err)
	assert.Equal(t, "after blanks", ev.Data)
}
