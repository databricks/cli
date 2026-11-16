package postgrescmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckSingleStatement(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "single statement", input: "SELECT 1", wantErr: false},
		{name: "trailing semicolon allowed", input: "SELECT 1;", wantErr: false},
		{name: "trailing semicolon plus whitespace", input: "SELECT 1;\n  ", wantErr: false},
		{name: "two statements rejected", input: "SELECT 1; SELECT 2", wantErr: true},
		{name: "two statements with trailing semi", input: "SELECT 1; SELECT 2;", wantErr: true},

		{name: "semicolon in single-quoted string", input: "SELECT 'a;b'", wantErr: false},
		{name: "semicolon in double-quoted ident", input: `SELECT "col;name" FROM t`, wantErr: false},
		{name: "doubled quote escape", input: "SELECT 'it''s;ok'", wantErr: false},
		{name: "doubled identifier quote", input: `SELECT "x""y;z" FROM t`, wantErr: false},

		{name: "semicolon in line comment", input: "SELECT 1 -- x;y\n", wantErr: false},
		{name: "semicolon in block comment", input: "SELECT 1 /* x;y */", wantErr: false},
		{name: "block comment unterminated", input: "SELECT 1 /* unterminated", wantErr: false},

		{name: "semicolon in dollar body untagged", input: "SELECT $$a;b$$", wantErr: false},
		{name: "semicolon in dollar body tagged", input: "SELECT $tag$a;b$tag$", wantErr: false},
		{name: "create function with body", input: "CREATE FUNCTION f() RETURNS int AS $$ BEGIN; END $$ LANGUAGE plpgsql", wantErr: false},

		{name: "semi inside string then real semi", input: "SELECT 'a;b'; SELECT 2", wantErr: true},
		{name: "semi inside line comment then real semi", input: "SELECT 1 -- ; \n; SELECT 2", wantErr: true},
		{name: "semi inside dollar then real semi", input: "SELECT $$a;b$$; SELECT 2", wantErr: true},

		{name: "trailing line comment after semi", input: "SELECT 1; -- done", wantErr: false},
		{name: "trailing block comment after semi", input: "SELECT 1; /* done */\n", wantErr: false},
		{name: "comment line after semi", input: "SELECT 1;\n-- end\n", wantErr: false},
		{name: "comment then second statement", input: "SELECT 1; -- x\nSELECT 2", wantErr: true},
		{name: "block comment then second statement", input: "SELECT 1; /* x */ SELECT 2", wantErr: true},
		{name: "CR ends line comment after semi", input: "SELECT 1; -- x\rSELECT 2", wantErr: true},
		{name: "CR ends line comment before semi", input: "SELECT 1 -- x\r; SELECT 2", wantErr: true},
		{name: "repeated trailing semicolons", input: "SELECT 1;;", wantErr: false},
		{name: "spaced trailing semicolons", input: "SELECT 1; ;", wantErr: false},
		{name: "semicolon after trailing comment", input: "SELECT 1; -- a\n;", wantErr: false},
		{name: "form feed after semi", input: "SELECT 1;\f", wantErr: false},
		{name: "nested block comment after semi", input: "SELECT 1; /* /* */ */", wantErr: false},
		{name: "nested block comment hides line comment after semi", input: "SELECT 1; /* /* */--*/ SELECT 2", wantErr: true},
		{name: "nested block comment hides line comment before semi", input: "SELECT 1 /* /* */--*/; SELECT 2", wantErr: true},
		{name: "semicolon in nested block comment", input: "SELECT 1 /* a /* ; */ ; */", wantErr: false},

		{name: "leading whitespace", input: "  ;", wantErr: false},
		{name: "empty input", input: "", wantErr: false},
		{name: "only whitespace", input: "  \n\t  ", wantErr: false},
		{name: "only semicolon", input: ";", wantErr: false},

		// $1 / $2 placeholder syntax must not be confused with a dollar-quote
		// tag (tags can't start with a digit per PG docs).
		{name: "dollar-digit placeholders", input: "SELECT $1, $2 FROM t", wantErr: false},
		{name: "dollar-digit then real semi", input: "SELECT $1 FROM t; SELECT 2", wantErr: true},

		// Tag must be an unquoted identifier. Punctuation rejects the
		// candidate so the embedded ';' is NOT hidden inside a fake body.
		{name: "dollar-tag with hyphen rejected", input: "SELECT $foo-bar$a;b$foo-bar$", wantErr: true},
		{name: "dollar-tag with dot rejected", input: "SELECT $foo.bar$a;b$foo.bar$", wantErr: true},
		{name: "dollar-tag with underscore", input: "SELECT $body_v2$a;b$body_v2$", wantErr: false},
		{name: "dollar-tag mixed letters digits", input: "SELECT $tag1$a;b$tag1$", wantErr: false},

		// E-string escape syntax: scanner doesn't honour \' escape, so a
		// backslash-escaped apostrophe terminates the literal early. We
		// document the over-rejection rather than fix it (acceptable v1
		// stance per the plan); pin the behaviour here so the next person
		// touching the scanner has to update the test.
		{name: "E-string with backslash-escape over-rejects", input: `SELECT E'foo\';bar'`, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := checkSingleStatement(tc.input)
			if tc.wantErr {
				assert.ErrorIs(t, err, errMultipleStatements)
				return
			}
			assert.NoError(t, err)
		})
	}
}
