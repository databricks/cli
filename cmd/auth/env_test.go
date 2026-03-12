package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuoteEnvValue(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "simple value", in: "hello", want: "hello"},
		{name: "empty value", in: "", want: `""`},
		{name: "value with space", in: "hello world", want: `"hello world"`},
		{name: "value with tab", in: "hello\tworld", want: "\"hello\tworld\""},
		{name: "value with double quote", in: `say "hi"`, want: `"say \"hi\""`},
		{name: "value with backslash", in: `path\to`, want: `"path\\to"`},
		{name: "url value", in: "https://example.com", want: "https://example.com"},
		{name: "value with dollar", in: "price$5", want: `"price$5"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := quoteEnvValue(c.in)
			assert.Equal(t, c.want, got)
		})
	}
}
