package main

import (
	"go/parser"
	"go/token"
	"slices"
	"strconv"
	"testing"
)

// importsOf re-parses formatted source and returns its import paths, asserting
// the result is valid Go (formatSource must never emit something that won't
// parse).
func importsOf(t *testing.T, src []byte) []string {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatalf("formatSource produced invalid Go: %v\n%s", err, src)
	}
	var paths []string
	for _, imp := range f.Imports {
		p, _ := strconv.Unquote(imp.Path.Value)
		paths = append(paths, p)
	}
	return paths
}

func TestFormatSourcePrunesUnusedImports(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "drops unused, keeps selector-used",
			in: `package p

import (
	"bytes"
	"fmt"
)

var _ = fmt.Sprint
`,
			want: []string{"fmt"},
		},
		{
			name: "keeps import used only as a type qualifier",
			in: `package p

import "bytes"

var _ bytes.Buffer
`,
			want: []string{"bytes"},
		},
		{
			name: "keeps aliased import used by alias, drops unused alias",
			in: `package p

import (
	keep "bytes"
	drop "strings"
)

var _ = keep.NewBuffer
`,
			want: []string{"bytes"},
		},
		{
			// The reason every template import carries an explicit alias when its
			// path base collides: prune keys on the referenced name, not the path.
			name: "distinguishes packages sharing a path base via alias",
			in: `package p

import (
	"time"
	sdktime "example.test/sdk/time"
)

var _ = sdktime.Now
`,
			want: []string{"example.test/sdk/time"},
		},
		{
			name: "drops an entire unused group",
			in: `package p

import (
	"bytes"
	"fmt"
	"strings"
)
`,
			want: nil,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := formatSource([]byte(c.in))
			if err != nil {
				t.Fatalf("formatSource: %v", err)
			}
			if got := importsOf(t, out); !slices.Equal(got, c.want) {
				t.Errorf("imports = %v, want %v\noutput:\n%s", got, c.want, out)
			}
		})
	}
}

func TestFormatSourceRejectsInvalidGo(t *testing.T) {
	if _, err := formatSource([]byte("package p\nfunc (")); err == nil {
		t.Fatal("expected an error for unparseable input, got nil")
	}
}

func TestImportName(t *testing.T) {
	src := `package p

import (
	"encoding/json"
	"github.com/databricks/cli/libs/cmdio"
	sdktime "github.com/databricks/databricks-sdk-go/common/types/time"
)
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"json", "cmdio", "sdktime"}
	for i, imp := range f.Imports {
		if got := importName(imp); got != want[i] {
			t.Errorf("importName(%s) = %q, want %q", imp.Path.Value, got, want[i])
		}
	}
}
