package interpolation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseEmpty(t *testing.T) {
	tokens, err := Parse("")
	require.NoError(t, err)
	assert.Nil(t, tokens)
}

func TestParseLiteralOnly(t *testing.T) {
	tokens, err := Parse("hello world")
	require.NoError(t, err)
	assert.Equal(t, []Token{
		{Kind: TokenLiteral, Value: "hello world", Start: 0, End: 11},
	}, tokens)
}

func TestParseSingleRef(t *testing.T) {
	tokens, err := Parse("${a.b}")
	require.NoError(t, err)
	assert.Equal(t, []Token{
		{Kind: TokenRef, Value: "a.b", Start: 0, End: 6},
	}, tokens)
}

func TestParseMultipleRefs(t *testing.T) {
	tokens, err := Parse("${a} ${b}")
	require.NoError(t, err)
	assert.Equal(t, []Token{
		{Kind: TokenRef, Value: "a", Start: 0, End: 4},
		{Kind: TokenLiteral, Value: " ", Start: 4, End: 5},
		{Kind: TokenRef, Value: "b", Start: 5, End: 9},
	}, tokens)
}

func TestParseMixedLiteralAndRef(t *testing.T) {
	tokens, err := Parse("pre ${a.b} post")
	require.NoError(t, err)
	assert.Equal(t, []Token{
		{Kind: TokenLiteral, Value: "pre ", Start: 0, End: 4},
		{Kind: TokenRef, Value: "a.b", Start: 4, End: 10},
		{Kind: TokenLiteral, Value: " post", Start: 10, End: 15},
	}, tokens)
}

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

func TestParseDollarAtEnd(t *testing.T) {
	tokens, err := Parse("abc$")
	require.NoError(t, err)
	assert.Equal(t, []Token{
		{Kind: TokenLiteral, Value: "abc$", Start: 0, End: 4},
	}, tokens)
}

func TestParseDollarBeforeNonBrace(t *testing.T) {
	tokens, err := Parse("$x")
	require.NoError(t, err)
	assert.Equal(t, []Token{
		{Kind: TokenLiteral, Value: "$x", Start: 0, End: 2},
	}, tokens)
}

func TestParseBackslashAtEnd(t *testing.T) {
	tokens, err := Parse(`abc\`)
	require.NoError(t, err)
	assert.Equal(t, []Token{
		{Kind: TokenLiteral, Value: `abc\`, Start: 0, End: 4},
	}, tokens)
}

func TestParseNestedReferenceReturnsError(t *testing.T) {
	_, err := Parse("${var.foo_${var.tail}}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nested variable references are not supported")
}

func TestParseUnterminatedRef(t *testing.T) {
	_, err := Parse("${a.b")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unterminated")
}

func TestParseEmptyRef(t *testing.T) {
	_, err := Parse("${}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty")
}

func TestParseInvalidPaths(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"trailing_hyphen", "${foo.bar-}"},
		{"double_dot", "${foo..bar}"},
		{"leading_digit", "${0foo}"},
		{"hyphen_start_segment", "${foo.-bar}"},
		{"trailing_dot", "${foo.}"},
		{"leading_dot", "${.foo}"},
		{"space_in_path", "${foo. bar}"},
		{"special_char", "${foo.bar!}"},
		{"just_digits", "${123}"},
		{"trailing_underscore", "${foo.bar_}"},
		{"underscore_start_segment", "${foo._bar}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.input)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid")
		})
	}
}

func TestParsePositions(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		tokens []Token
	}{
		{
			"single_ref",
			"${a.b}",
			[]Token{{Kind: TokenRef, Value: "a.b", Start: 0, End: 6}},
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
			"escaped_ref",
			`\${a}`,
			[]Token{{Kind: TokenLiteral, Value: "${a}", Start: 0, End: 5}},
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
			"dollar_sign_mid_literal",
			"a$b",
			[]Token{{Kind: TokenLiteral, Value: "a$b", Start: 0, End: 3}},
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
