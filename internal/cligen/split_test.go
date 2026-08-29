package main

import (
	"reflect"
	"testing"
)

// TestSplitASCII pins the exact behavior of splitASCII (names.go), the engine
// behind every casing function. splitASCII is a faithful port of genkit's
// Named.splitASCII, which emulates a JVM lookahead/lookbehind regex by scanning
// for the nearest letter in both directions. These cases document the
// non-obvious consequences of that emulation, which a simpler reimplementation
// (e.g. Go's lookaround-free regexp) would not reproduce.
func TestSplitASCII(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		// Empty and separator-only inputs yield no words.
		{"", nil},
		{"_", nil},
		{"__", nil},

		// '$' is dropped entirely, joining its neighbors.
		{"foo$bar", []string{"foobar"}},
		{"$foo", []string{"foo"}},

		// Single words, any casing, lowercased output.
		{"a", []string{"a"}},
		{"A", []string{"a"}},
		{"AB", []string{"ab"}},
		{"ABC", []string{"abc"}},
		{"Abc", []string{"abc"}},
		{"Workspace", []string{"workspace"}},

		// Plain camel/Pascal and separator splits.
		{"JobSettings", []string{"job", "settings"}},
		{"IpAccessLists", []string{"ip", "access", "lists"}},
		{"create_run", []string{"create", "run"}},
		{"snake_case_name", []string{"snake", "case", "name"}},

		// Acronym tail: the last capital of a run starts the next word when it is
		// followed by a lowercase letter.
		{"HTTPSConnection", []string{"https", "connection"}},
		{"ServeMLModel", []string{"serve", "ml", "model"}},
		{"ABCDef", []string{"abc", "def"}},
		{"GetByID", []string{"get", "by", "id"}},
		{"DefABC", []string{"def", "abc"}},

		// Acronym head quirk: a leading capital splits off alone when the run is
		// exactly two capitals followed by lowercase (the lookahead can't see far
		// enough to keep them together)...
		{"ABc", []string{"a", "bc"}},
		{"AItool", []string{"a", "itool"}},
		{"MLflow", []string{"m", "lflow"}},
		// ...but three+ capitals before lowercase keep the run intact.
		{"AIService", []string{"ai", "service"}},

		// Digits attach to the preceding word; they never start a word.
		{"IamV2", []string{"iam", "v2"}},
		{"AccountGroupsV2", []string{"account", "groups", "v2"}},
		{"HTTP2Server", []string{"http2", "server"}},
		{"OAuth2Config", []string{"o", "auth2", "config"}},
		{"UUID4", []string{"uuid4"}},
		{"a1b2", []string{"a1b2"}},
	}
	for _, c := range cases {
		if got := splitASCII(c.in); !reflect.DeepEqual(got, c.want) {
			t.Errorf("splitASCII(%q) = %#v, want %#v", c.in, got, c.want)
		}
	}
}
