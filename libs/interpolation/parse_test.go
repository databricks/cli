package interpolation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseValidPaths(t *testing.T) {
	tests := []struct {
		input string
		path  string
	}{
		{"${a}", "a"},
		{"${abc}", "abc"},
		{"${a.b.c}", "a.b.c"},
		{"${a.b[0]}", "a.b[0]"},
		{"${a[0]}", "a[0]"},
		{"${a.b[0][1]}", "a.b[0][1]"},
		{"${a.b-c}", "a.b-c"},
		{"${a.b_c}", "a.b_c"},
		{"${a.b-c-d}", "a.b-c-d"},
		{"${a.b_c_d}", "a.b_c_d"},
		{"${abc.def.ghi}", "abc.def.ghi"},
		{"${a.b123}", "a.b123"},
		{"${resources.jobs.my-job.id}", "resources.jobs.my-job.id"},
		{"${var.my_var}", "var.my_var"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens, err := Parse(tt.input)
			require.NoError(t, err)
			require.Len(t, tokens, 1)
			assert.Equal(t, TokenRef, tokens[0].Kind)
			assert.Equal(t, tt.path, tokens[0].Value)
		})
	}
}

func TestParseEscapeSequences(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		tokens []Token
	}{
		{
			"escaped_dollar",
			`\$`,
			[]Token{{Kind: TokenLiteral, Value: "$", Start: 0, End: 2}},
		},
		{
			"escaped_ref",
			`\${a}`,
			[]Token{{Kind: TokenLiteral, Value: "${a}", Start: 0, End: 5}},
		},
		{
			"escaped_backslash",
			`\\`,
			[]Token{{Kind: TokenLiteral, Value: `\`, Start: 0, End: 2}},
		},
		{
			"double_escaped_backslash",
			`\\\\`,
			[]Token{{Kind: TokenLiteral, Value: `\\`, Start: 0, End: 4}},
		},
		{
			"escaped_backslash_then_ref",
			`\\${a.b}`,
			[]Token{
				{Kind: TokenLiteral, Value: `\`, Start: 0, End: 2},
				{Kind: TokenRef, Value: "a.b", Start: 2, End: 8},
			},
		},
		{
			"backslash_before_non_special",
			`\n`,
			[]Token{{Kind: TokenLiteral, Value: `\n`, Start: 0, End: 2}},
		},
		{
			"escaped_dollar_then_ref",
			`\$\$${a.b}`,
			[]Token{
				{Kind: TokenLiteral, Value: "$$", Start: 0, End: 4},
				{Kind: TokenRef, Value: "a.b", Start: 4, End: 10},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.tokens, tokens)
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		tokens []Token
	}{
		{
			"empty",
			"",
			nil,
		},
		{
			"literal_only",
			"hello world",
			[]Token{{Kind: TokenLiteral, Value: "hello world", Start: 0, End: 11}},
		},
		{
			"single_ref",
			"${a.b}",
			[]Token{{Kind: TokenRef, Value: "a.b", Start: 0, End: 6}},
		},
		{
			"multiple_refs",
			"${a} ${b}",
			[]Token{
				{Kind: TokenRef, Value: "a", Start: 0, End: 4},
				{Kind: TokenLiteral, Value: " ", Start: 4, End: 5},
				{Kind: TokenRef, Value: "b", Start: 5, End: 9},
			},
		},
		{
			"literal_ref_literal",
			"pre ${a.b} post",
			[]Token{
				{Kind: TokenLiteral, Value: "pre ", Start: 0, End: 4},
				{Kind: TokenRef, Value: "a.b", Start: 4, End: 10},
				{Kind: TokenLiteral, Value: " post", Start: 10, End: 15},
			},
		},
		{
			"adjacent_refs",
			"${a}${b}",
			[]Token{
				{Kind: TokenRef, Value: "a", Start: 0, End: 4},
				{Kind: TokenRef, Value: "b", Start: 4, End: 8},
			},
		},
		{
			"dollar_at_end",
			"abc$",
			[]Token{{Kind: TokenLiteral, Value: "abc$", Start: 0, End: 4}},
		},
		{
			"dollar_before_non_brace",
			"$x",
			[]Token{{Kind: TokenLiteral, Value: "$x", Start: 0, End: 2}},
		},
		{
			"dollar_mid_literal",
			"a$b",
			[]Token{{Kind: TokenLiteral, Value: "a$b", Start: 0, End: 3}},
		},
		{
			"backslash_at_end",
			`abc\`,
			[]Token{{Kind: TokenLiteral, Value: `abc\`, Start: 0, End: 4}},
		},
		{
			"escaped_ref",
			`\${a}`,
			[]Token{{Kind: TokenLiteral, Value: "${a}", Start: 0, End: 5}},
		},
		{
			"escape_between_refs",
			`${a}\$${b}`,
			[]Token{
				{Kind: TokenRef, Value: "a", Start: 0, End: 4},
				{Kind: TokenLiteral, Value: "$", Start: 4, End: 6},
				{Kind: TokenRef, Value: "b", Start: 6, End: 10},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Parse(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.tokens, tokens)
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		errContains string
	}{
		{"nested_reference", "${var.foo_${var.tail}}", "nested variable references are not supported"},
		{"unterminated_ref", "${a.b", "unterminated"},
		{"empty_ref", "${}", "empty"},
		{"trailing_hyphen", "${foo.bar-}", "invalid"},
		{"double_dot", "${foo..bar}", "invalid"},
		{"leading_digit", "${0foo}", "invalid"},
		{"hyphen_start_segment", "${foo.-bar}", "invalid"},
		{"trailing_dot", "${foo.}", "invalid"},
		{"leading_dot", "${.foo}", "invalid"},
		{"space_in_path", "${foo. bar}", "invalid"},
		{"special_char", "${foo.bar!}", "invalid"},
		{"just_digits", "${123}", "invalid"},
		{"trailing_underscore", "${foo.bar_}", "invalid"},
		{"underscore_start_segment", "${foo._bar}", "invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errContains)
		})
	}
}
