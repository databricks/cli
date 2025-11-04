package cmdio

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompat_readLine(t *testing.T) {
	tests := []struct {
		name       string
		reader     io.Reader
		want       string
		wantErr    bool
		errContain string
	}{
		{
			name:   "basic line with LF",
			reader: strings.NewReader("hello\n"),
			want:   "hello",
		},
		{
			name:   "line with CRLF",
			reader: strings.NewReader("hello\r\n"),
			want:   "hello",
		},
		{
			name:   "line with only CR",
			reader: strings.NewReader("hello\r"),
			want:   "hello",
		},
		{
			name:   "empty line with LF",
			reader: strings.NewReader("\n"),
			want:   "",
		},
		{
			name:   "empty line with CRLF",
			reader: strings.NewReader("\r\n"),
			want:   "",
		},
		{
			name:   "line without newline at EOF",
			reader: strings.NewReader("hello"),
			want:   "hello",
		},
		{
			name:   "multi-line input stops at first newline",
			reader: strings.NewReader("hello\nworld\n"),
			want:   "hello",
		},
		{
			name:   "line with multiple CR characters",
			reader: strings.NewReader("hello\r\rworld\n"),
			want:   "helloworld",
		},
		{
			name:   "line with CR in middle",
			reader: strings.NewReader("hello\rworld\n"),
			want:   "helloworld",
		},
		{
			name:   "line with spaces and special chars",
			reader: strings.NewReader("hello world!@#$%\n"),
			want:   "hello world!@#$%",
		},
		{
			name:   "unicode characters",
			reader: strings.NewReader("hello 世界\n"),
			want:   "hello 世界",
		},
		{
			name:       "empty reader",
			reader:     strings.NewReader(""),
			want:       "",
			wantErr:    true,
			errContain: "EOF",
		},
		{
			name:       "reader with error",
			reader:     &errorReader{err: errors.New("read error")},
			want:       "",
			wantErr:    true,
			errContain: "read error",
		},
		{
			name:   "reader with error after some data",
			reader: &errorAfterNReader{data: []byte("hello"), n: 3, err: io.ErrUnexpectedEOF},
			want:   "hel",
		},
		{
			name:   "long line",
			reader: strings.NewReader(strings.Repeat("a", 1000) + "\n"),
			want:   strings.Repeat("a", 1000),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := readLine(test.reader)
			if test.wantErr {
				require.Error(t, err)
				if test.errContain != "" {
					assert.Contains(t, err.Error(), test.errContain)
				}
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, test.want, got)
		})
	}
}

// errorReader is a test helper that always returns an error
type errorReader struct {
	err error
}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, e.err
}

// errorAfterNReader is a test helper that returns data for n reads, then returns an error
type errorAfterNReader struct {
	data  []byte
	n     int
	count int
	err   error
}

func (e *errorAfterNReader) Read(p []byte) (n int, err error) {
	if e.count >= e.n {
		return 0, e.err
	}
	if len(p) == 0 {
		return 0, nil
	}
	if e.count < len(e.data) {
		p[0] = e.data[e.count]
		e.count++
		return 1, nil
	}
	return 0, e.err
}

func TestCompat_splitAtLastNewLine(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantFirst string
		wantLast  string
	}{
		{
			name:      "LF newline in middle",
			input:     "hello\nworld",
			wantFirst: "hello\n",
			wantLast:  "world",
		},
		{
			name:      "CRLF newline in middle",
			input:     "hello\r\nworld",
			wantFirst: "hello\r\n",
			wantLast:  "world",
		},
		{
			name:      "no newline",
			input:     "hello world",
			wantFirst: "",
			wantLast:  "hello world",
		},
		{
			name:      "newline at end",
			input:     "hello\nworld\n",
			wantFirst: "hello\nworld\n",
			wantLast:  "",
		},
		{
			name:      "newline at start",
			input:     "\nhello world",
			wantFirst: "\n",
			wantLast:  "hello world",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first, last := splitAtLastNewLine(tt.input)
			assert.Equal(t, tt.wantFirst, first)
			assert.Equal(t, tt.wantLast, last)
		})
	}
}
