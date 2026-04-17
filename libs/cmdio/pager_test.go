package cmdio

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCrlfWriterTranslatesNewlines(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no newlines", "abc", "abc"},
		{"single newline mid", "a\nb", "a\r\nb"},
		{"newline at end", "abc\n", "abc\r\n"},
		{"newline at start", "\nabc", "\r\nabc"},
		{"consecutive newlines", "\n\n", "\r\n\r\n"},
		{"multiple lines", "one\ntwo\nthree\n", "one\r\ntwo\r\nthree\r\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := crlfWriter{w: &buf}
			n, err := w.Write([]byte(tt.in))
			require.NoError(t, err)
			assert.Equal(t, len(tt.in), n, "Write must return the input byte count")
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestPagerShouldQuitDrainsNonQuitKeys(t *testing.T) {
	ch := make(chan byte, 4)
	ch <- ' '
	ch <- 'x'
	ch <- 'y'
	assert.False(t, pagerShouldQuit(ch), "non-quit keys must return false")
	assert.Empty(t, ch, "non-quit keys must be drained from the channel")
}

func TestPagerShouldQuitReturnsTrueForQuitKeys(t *testing.T) {
	for _, k := range []byte{'q', 'Q', pagerKeyEscape, pagerKeyCtrlC} {
		ch := make(chan byte, 1)
		ch <- k
		assert.Truef(t, pagerShouldQuit(ch), "key %q must trigger quit", k)
	}
}

func TestPagerShouldQuitClosedChannelKeepsDraining(t *testing.T) {
	ch := make(chan byte)
	close(ch)
	assert.False(t, pagerShouldQuit(ch), "closed channel (stdin EOF) must not force quit")
}

func TestPagerNextKeyReturnsFalseOnClosedChannel(t *testing.T) {
	ch := make(chan byte)
	close(ch)
	_, ok := pagerNextKey(t.Context(), ch)
	assert.False(t, ok)
}

func TestPagerNextKeyReturnsKey(t *testing.T) {
	ch := make(chan byte, 1)
	ch <- ' '
	k, ok := pagerNextKey(t.Context(), ch)
	assert.True(t, ok)
	assert.Equal(t, byte(' '), k)
}
