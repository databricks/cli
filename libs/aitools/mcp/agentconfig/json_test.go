package agentconfig

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/aitools/mcpconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tailscale/hujson"
)

// semanticJSONC decodes a file with comments and trailing commas.
func semanticJSONC(t *testing.T, data string) map[string]any {
	t.Helper()
	std, err := hujson.Standardize([]byte(data))
	require.NoError(t, err)
	return semantic(t, mcpconfig.FormatJSON, std)
}

// jsonEdit runs fn on a document parsed from data and returns the result bytes.
func jsonEdit(t *testing.T, data string, fn func(d document)) string {
	t.Helper()
	doc, err := parseJSON([]byte(data))
	require.NoError(t, err)
	fn(doc)
	out, err := doc.bytes()
	require.NoError(t, err)
	return string(out)
}

func TestJSONKeepsCommentsTrailingCommasAndValues(t *testing.T) {
	data := `{
  // Agent settings
  "theme": "dark", /* keep */
  "big": 12345678901234567890, "text": "café",
  "mcpServers": {
    "other": {"command": "npx"},
  },
}
`
	entry := testEntry(testCmd, testArgs("test"), nil)
	got := jsonEdit(t, data, func(d document) {
		require.NoError(t, d.put([]string{"mcpServers"}, testName, entry, nil))
	})
	assert.Contains(t, got, "// Agent settings\n")
	assert.Contains(t, got, `"theme": "dark", /* keep */`)
	assert.Contains(t, got, `"big": 12345678901234567890, "text": "café",`)
	assert.Contains(t, got, "\"other\": {\"command\": \"npx\"},\n    \""+testName+"\": {\n")
	assert.Contains(t, got, "\n  },\n}\n")

	// Removing the entry leaves the other members and the comments in place.
	back := jsonEdit(t, got, func(d document) {
		require.NoError(t, d.put([]string{"mcpServers"}, testName, nil, nil))
	})
	assert.Contains(t, back, "// Agent settings\n")
	assert.Equal(t, semanticJSONC(t, data), semanticJSONC(t, back))
}

func TestJSONCreatesMissingLevels(t *testing.T) {
	got := jsonEdit(t, `{"theme": "dark"}`, func(d document) {
		require.NoError(t, d.put([]string{"mcp", "servers"}, "x", map[string]any{"type": "local"}, nil))
	})
	assert.Equal(t, map[string]any{
		"theme": "dark",
		"mcp":   map[string]any{"servers": map[string]any{"x": map[string]any{"type": "local"}}},
	}, semantic(t, mcpconfig.FormatJSON, []byte(got)))
}

func TestJSONBOMIsKept(t *testing.T) {
	got := jsonEdit(t, "\xEF\xBB\xBF{\"a\": 1}\n", func(d document) {
		require.NoError(t, d.put([]string{"mcpServers"}, testName, testEntry(testCmd, nil, nil), nil))
	})
	assert.Equal(t, "\xEF\xBB\xBF", got[:3])
	assert.Contains(t, got, `"a": 1`)
	assert.Contains(t, got, testName)
}

// TestJSONNewFileLayout checks that a new file gets two-space indentation and a
// trailing newline, and <, > and & are written as they are.
func TestJSONNewFileLayout(t *testing.T) {
	entry := map[string]any{"command": "/a<b>&c", "args": []string{"x"}}
	got := jsonEdit(t, "", func(d document) {
		require.NoError(t, d.put([]string{"mcpServers"}, "x", entry, nil))
	})
	assert.Equal(t, `{
  "mcpServers": {
    "x": {
      "args": [
        "x"
      ],
      "command": "/a<b>&c"
    }
  }
}
`, got)
}

func TestJSONRejectsNonObjectTopLevel(t *testing.T) {
	_, err := parseJSON([]byte(`"text"`))
	require.ErrorContains(t, err, "top-level value is not a JSON object")
}

func TestJSONEntriesSkipNonObjects(t *testing.T) {
	doc, err := parseJSON([]byte(`{"mcpServers": {"a": {"command": "x"}, "b": 3}}`))
	require.NoError(t, err)
	entries, err := doc.entries([]string{"mcpServers"})
	require.NoError(t, err)
	assert.Equal(t, map[string]map[string]any{"a": {"command": "x"}}, entries)
}

// requireStrictJSON fails unless data is strict JSON, which agents such as
// Claude Code read.
func requireStrictJSON(t *testing.T, data string) {
	t.Helper()
	var v any
	require.NoError(t, json.Unmarshal([]byte(data), &v), data)
}

// TestJSONAddKeepsCommentOnPreviousMember checks that a comment after the last
// member stays on that member's line when a member is added after it, in LF and
// CRLF files.
func TestJSONAddKeepsCommentOnPreviousMember(t *testing.T) {
	for _, nl := range []string{"\n", "\r\n"} {
		data := strings.ReplaceAll("{\n  \"mcpServers\": {\n    \"other\": {\"command\": \"npx\"} // other server\n  }\n}\n", "\n", nl)
		got := jsonEdit(t, data, func(d document) {
			require.NoError(t, d.put([]string{"mcpServers"}, "x", map[string]any{"command": "x"}, nil))
		})
		want := strings.ReplaceAll("{\n  \"mcpServers\": {\n    \"other\": {\"command\": \"npx\"}, // other server\n    \"x\": {\n      \"command\": \"x\"\n    }\n  }\n}\n", "\n", nl)
		assert.Equal(t, want, got, "line ending %q", nl)
	}
}

// TestJSONAddToEmptyTable checks that an empty table gets its member on its own
// line, and its closing brace on a line of its own.
func TestJSONAddToEmptyTable(t *testing.T) {
	got := jsonEdit(t, "{\n  \"mcpServers\": {}\n}\n", func(d document) {
		require.NoError(t, d.put([]string{"mcpServers"}, "x", map[string]any{"command": "x"}, nil))
	})
	assert.Equal(t, "{\n  \"mcpServers\": {\n    \"x\": {\n      \"command\": \"x\"\n    }\n  }\n}\n", got)
}

// TestJSONAddKeepsCommentsOfEmptyObject checks that a file with only comments and
// an empty object keeps its comments. Only the layout of the object changes.
func TestJSONAddKeepsCommentsOfEmptyObject(t *testing.T) {
	for _, tc := range []struct {
		data string
		want string
	}{
		{"// x\n{}", "// x\n{\n  \"mcpServers\": {\n    \"x\": {\n      \"command\": \"x\"\n    }\n  }\n}"},
		{"{ /* c */ }", "{\n  \"mcpServers\": {\n    \"x\": {\n      \"command\": \"x\"\n    }\n  } /* c */\n}"},
	} {
		t.Run(tc.data, func(t *testing.T) {
			got := jsonEdit(t, tc.data, func(d document) {
				require.NoError(t, d.put([]string{"mcpServers"}, "x", map[string]any{"command": "x"}, nil))
			})
			assert.Equal(t, tc.want, got)
		})
	}
}

// TestJSONAddToEmptyObjectFile checks that an empty object file gets one trailing
// newline, as the file had.
func TestJSONAddToEmptyObjectFile(t *testing.T) {
	want := "{\n  \"mcpServers\": {\n    \"x\": {\n      \"command\": \"x\"\n    }\n  }\n}\n"
	for _, data := range []string{"{}\n", "", " \n"} {
		got := jsonEdit(t, data, func(d document) {
			require.NoError(t, d.put([]string{"mcpServers"}, "x", map[string]any{"command": "x"}, nil))
		})
		assert.Equal(t, want, got, "input %q", data)
	}
}

// TestJSONRemoveLastKeepsStrictJSON checks that removing the last member keeps a
// strict file strict. The comma that stood after the previous member's space is
// not left behind.
func TestJSONRemoveLastKeepsStrictJSON(t *testing.T) {
	for _, data := range []string{
		"{\n  \"mcpServers\": {\n    \"a\": {} ,\n    \"b\": {\"command\": \"x\"}\n  }\n}\n",
		"{\"mcpServers\": {\"a\": {} , \"b\": {\"command\": \"x\"}}}",
		"{\"mcpServers\": {\"a\": {}  , \"b\": {\"command\": \"x\"}}}",
	} {
		got := jsonEdit(t, data, func(d document) {
			require.NoError(t, d.put([]string{"mcpServers"}, "b", nil, nil))
		})
		requireStrictJSON(t, got)
		assert.Equal(t, map[string]any{"mcpServers": map[string]any{"a": map[string]any{}}}, semantic(t, mcpconfig.FormatJSON, []byte(got)))
	}
}

// TestJSONRemoveLastKeepsTrailingComma checks that a file with a trailing comma
// keeps the trailing comma when its last member is removed.
func TestJSONRemoveLastKeepsTrailingComma(t *testing.T) {
	got := jsonEdit(t, "{\"mcpServers\": {\"a\": {}, \"b\": {},}}", func(d document) {
		require.NoError(t, d.put([]string{"mcpServers"}, "b", nil, nil))
	})
	assert.Equal(t, "{\"mcpServers\": {\"a\": {},}}", got)
}

// TestJSONKeepsCRLF checks that a CRLF file gets CRLF in inserted text, so the
// file has one line ending.
func TestJSONKeepsCRLF(t *testing.T) {
	data := "{\r\n  \"mcpServers\": {\r\n    \"other\": {\"command\": \"npx\"}\r\n  }\r\n}\r\n"
	got := jsonEdit(t, data, func(d document) {
		require.NoError(t, d.put([]string{"mcpServers"}, "x", map[string]any{"command": "x", "args": []string{"a"}}, nil))
	})
	assert.Equal(t, strings.Count(got, "\n"), strings.Count(got, "\r\n"), got)
	assert.Contains(t, got, "\"other\": {\"command\": \"npx\"},\r\n    \"x\": {\r\n")

	got = jsonEdit(t, got, func(d document) {
		require.NoError(t, d.put([]string{"mcpServers"}, "x", nil, nil))
	})
	assert.Equal(t, data, got)
}

// TestJSONAddKeepsStrictJSON checks that adding a member to a strict file keeps it
// strict, whatever its layout.
func TestJSONAddKeepsStrictJSON(t *testing.T) {
	for _, data := range []string{
		"{}",
		"{\n}\n",
		`{"mcpServers": {}}`,
		"{\n  \"mcpServers\": {\n    \"a\": {} , \"b\": {\"command\": \"x\"}\n  }\n}\n",
		"{\"theme\": \"dark\", \"mcpServers\": {\"a\": {\"command\": \"x\"}}}",
	} {
		got := jsonEdit(t, data, func(d document) {
			require.NoError(t, d.put([]string{"mcpServers"}, "x", map[string]any{"command": "y"}, nil))
		})
		requireStrictJSON(t, got)
		assert.Equal(t, "y", semantic(t, mcpconfig.FormatJSON, []byte(got))["mcpServers"].(map[string]any)["x"].(map[string]any)["command"], data)
	}
}

// TestJSONRemoveLastDropsEmptyContainers checks that removing the last member of
// a container drops the container, and each container above it that it leaves
// empty. A file gets back the bytes it had before the add, in LF and CRLF files.
func TestJSONRemoveLastDropsEmptyContainers(t *testing.T) {
	for _, tc := range []struct {
		name string
		root []string
		data string
	}{
		{"one level", []string{"mcpServers"}, "{\n  \"theme\": \"dark\"\n}\n"},
		{"one level compact", []string{"mcpServers"}, `{"theme": "dark"}`},
		{"one level before a member", []string{"mcpServers"}, "{\n  \"theme\": \"dark\",\n  \"model\": \"x\"\n}\n"},
		{"two levels", []string{"mcp", "servers"}, "{\n  \"theme\": \"dark\"\n}\n"},
		{"two levels compact", []string{"mcp", "servers"}, `{"theme": "dark"}`},
		{"empty file object", []string{"mcpServers"}, "{}\n"},
		{"sibling of the emptied level stays", []string{"mcp", "servers"}, `{"mcp": {"other": 1}}`},
	} {
		for _, nl := range []string{"\n", "\r\n"} {
			t.Run(tc.name, func(t *testing.T) {
				data := strings.ReplaceAll(tc.data, "\n", nl)
				added := jsonEdit(t, data, func(d document) {
					require.NoError(t, d.put(tc.root, "x", map[string]any{"command": "x"}, nil))
				})
				got := jsonEdit(t, added, func(d document) {
					require.NoError(t, d.put(tc.root, "x", nil, nil))
				})
				assert.Equal(t, data, got, "line ending %q", nl)
			})
		}
	}
}

// TestJSONCheckEditAcceptsOnlyTheEdit checks the check after a JSON edit. An edit
// that changes another setting, a sibling, or the server's own value is refused.
// A removal may drop the containers it leaves empty, up to the root, and nothing else.
func TestJSONCheckEditAcceptsOnlyTheEdit(t *testing.T) {
	entry := testEntry(testCmd, testArgs("test"), nil)
	for _, tc := range []struct {
		name   string
		root   []string
		entry  map[string]any
		before map[string]any
		after  map[string]any
		want   string
	}{
		{
			name:   "add with an other setting changed",
			root:   []string{"mcpServers"},
			entry:  entry,
			before: map[string]any{"theme": "dark", "mcpServers": map[string]any{}},
			after:  map[string]any{"theme": "light", "mcpServers": map[string]any{testName: entry}},
			want:   "changed other settings",
		},
		{
			name:   "add that writes a different server",
			root:   []string{"mcpServers"},
			entry:  entry,
			before: map[string]any{"mcpServers": map[string]any{}},
			after:  map[string]any{"mcpServers": map[string]any{testName: map[string]any{"command": "other"}}},
			want:   "wrote a different server",
		},
		{
			name:   "add that does not write the server",
			root:   []string{"mcpServers"},
			entry:  entry,
			before: map[string]any{"mcpServers": map[string]any{}},
			after:  map[string]any{"mcpServers": map[string]any{}},
			want:   "did not add the server",
		},
		{
			name:   "removal that leaves the server",
			root:   []string{"mcpServers"},
			before: map[string]any{"mcpServers": map[string]any{testName: entry}},
			after:  map[string]any{"mcpServers": map[string]any{testName: entry}},
			want:   "did not remove the server",
		},
		{
			name:   "removal that also drops a sibling server",
			root:   []string{"mcpServers"},
			before: map[string]any{"mcpServers": map[string]any{testName: entry, "other": map[string]any{"command": "x"}}},
			after:  map[string]any{"mcpServers": map[string]any{}},
			want:   "changed other settings",
		},
		{
			name:   "removal that also drops an unrelated empty container",
			root:   []string{"mcpServers"},
			before: map[string]any{"mcpServers": map[string]any{testName: entry}, "extra": map[string]any{}},
			after:  map[string]any{},
			want:   "changed other settings",
		},
		{
			name:   "removal of the last server drops the root table",
			root:   []string{"mcpServers"},
			before: map[string]any{"theme": "dark", "mcpServers": map[string]any{testName: entry}},
			after:  map[string]any{"theme": "dark"},
		},
		{
			name:   "removal of the last server drops each container up to the root",
			root:   []string{"mcp", "servers"},
			before: map[string]any{"mcp": map[string]any{"servers": map[string]any{testName: entry}}},
			after:  map[string]any{},
		},
		{
			name:   "removal keeps a sibling of an emptied container",
			root:   []string{"mcp", "servers"},
			before: map[string]any{"mcp": map[string]any{"servers": map[string]any{testName: entry}, "other": 1.0}},
			after:  map[string]any{"mcp": map[string]any{"other": 1.0}},
		},
		{
			name:   "removal of an absent server changes nothing",
			root:   []string{"mcpServers"},
			before: map[string]any{"mcpServers": map[string]any{}},
			after:  map[string]any{"mcpServers": map[string]any{}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := checkEdit(tc.before, tc.after, tc.root, testName, tc.entry, sameShape, sameEntry)
			if tc.want == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tc.want)
		})
	}
}

// TestJSONCheckEditComparesOtherMembersAsBytes checks that a change to a member off
// the root path is refused even though the check does not decode that member.
func TestJSONCheckEditComparesOtherMembersAsBytes(t *testing.T) {
	parsed, err := parseJSON([]byte(`{"other":{"a":1},"mcpServers":{}}`))
	require.NoError(t, err)
	d := parsed.(*jsonDoc)
	root := []string{"mcpServers"}
	before := d.shape(root, testName)
	require.NoError(t, d.apply([]patch{{op: opReplace, path: "/other/a", value: []byte("2")}}))
	after := d.shape(root, testName)
	assert.ErrorContains(t, checkEdit(before, after, root, testName, nil, sameShape, sameEntry), "changed other settings")
}

// BenchmarkJSONAddBesideLargeMember times an add to a config whose other top-level
// member is large, as ~/.claude.json is, where the check must not decode that member.
// The file is parsed again for each run, outside the timer.
func BenchmarkJSONAddBesideLargeMember(b *testing.B) {
	entry := testEntry(testCmd, testArgs("test"), nil)
	for _, size := range []int{1 << 20, 16 << 20} {
		b.Run(fmt.Sprintf("%dMB", size>>20), func(b *testing.B) {
			data := []byte(`{"history":"` + strings.Repeat("x", size) + `","mcpServers":{}}`)
			b.ReportAllocs()
			for b.Loop() {
				b.StopTimer()
				doc, err := parseJSON(data)
				require.NoError(b, err)
				b.StartTimer()
				if err := doc.put([]string{"mcpServers"}, testName, entry, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
