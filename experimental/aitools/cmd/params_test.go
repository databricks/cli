package aitools

import (
	"testing"

	sql "github.com/databricks/sdk-go/statementexecution/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseParams(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []sql.StatementParameter
	}{
		{
			name: "nil input returns nil",
			in:   nil,
			want: nil,
		},
		{
			name: "empty input returns nil",
			in:   []string{},
			want: nil,
		},
		{
			name: "single string param defaults type to empty (server-side STRING)",
			in:   []string{"name=alice"},
			want: []sql.StatementParameter{
				{Name: new("name"), Value: new("alice")},
			},
		},
		{
			name: "typed param splits name and type on first colon",
			in:   []string{"since:DATE=2026-01-01"},
			want: []sql.StatementParameter{
				{Name: new("since"), Type: new("DATE"), Value: new("2026-01-01")},
			},
		},
		{
			name: "value can contain = and :",
			in:   []string{"clause=ts >= '2026-01-01T00:00:00'"},
			want: []sql.StatementParameter{
				{Name: new("clause"), Value: new("ts >= '2026-01-01T00:00:00'")},
			},
		},
		{
			name: "decimal type with parens preserved",
			in:   []string{"amount:DECIMAL(10,2)=42.50"},
			want: []sql.StatementParameter{
				{Name: new("amount"), Type: new("DECIMAL(10,2)"), Value: new("42.50")},
			},
		},
		{
			name: "empty value becomes NULL on the wire via omitempty",
			in:   []string{"opt="},
			want: []sql.StatementParameter{
				{Name: new("opt"), Value: new("")},
			},
		},
		{
			name: "whitespace around name and type is trimmed",
			in:   []string{"  name  :  INT  =42"},
			want: []sql.StatementParameter{
				{Name: new("name"), Type: new("INT"), Value: new("42")},
			},
		},
		{
			name: "multiple params preserve input order",
			in:   []string{"a=1", "b:INT=2", "c=three"},
			want: []sql.StatementParameter{
				{Name: new("a"), Value: new("1")},
				{Name: new("b"), Type: new("INT"), Value: new("2")},
				{Name: new("c"), Value: new("three")},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseParams(tc.in)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseParamsErrors(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		wantMsg string
	}{
		{
			name:    "no equals sign",
			in:      []string{"foo"},
			wantMsg: "expected name=value",
		},
		{
			name:    "empty name",
			in:      []string{"=value"},
			wantMsg: "name is empty",
		},
		{
			name:    "empty name with type",
			in:      []string{":INT=42"},
			wantMsg: "name is empty",
		},
		{
			name:    "whitespace-only name",
			in:      []string{"   =value"},
			wantMsg: "name is empty",
		},
		{
			name:    "duplicate name",
			in:      []string{"name=alice", "name=bob"},
			wantMsg: `duplicate --param name "name"`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseParams(tc.in)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantMsg)
		})
	}
}
