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

		{name: "leading whitespace", input: "  ;", wantErr: false},
		{name: "empty input", input: "", wantErr: false},
		{name: "only whitespace", input: "  \n\t  ", wantErr: false},
		{name: "only semicolon", input: ";", wantErr: false},
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
