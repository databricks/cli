package testdiff

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplacement_Literal(t *testing.T) {
	var repls ReplacementsContext

	repls.Set(`foobar`, `[replacement]`)
	assert.Equal(t, `[replacement]`, repls.Replace(`foobar`))
}

func TestReplacement_Encoded(t *testing.T) {
	var repls ReplacementsContext

	repls.Set(`foo"bar`, `[replacement]`)
	assert.Equal(t, `"[replacement]"`, repls.Replace(`"foo\"bar"`))
}

func TestReplacement_UUID(t *testing.T) {
	var repls ReplacementsContext

	PrepareReplacementsUUID(t, &repls)

	// A lone UUID stays unsuffixed.
	assert.Equal(t, "[UUID]", repls.Replace("123e4567-e89b-12d3-a456-426614174000"))
}

func TestReplacement_UUIDUnique(t *testing.T) {
	var repls ReplacementsContext

	PrepareReplacementsUUID(t, &repls)

	input := "a: 123e4567-e89b-12d3-a456-426614174000\n" +
		"b: 00000000-0000-0000-0000-000000000001\n" +
		"a again: 123e4567-e89b-12d3-a456-426614174000"
	expected := "a: [UUID][0]\n" +
		"b: [UUID][1]\n" +
		"a again: [UUID][0]"
	assert.Equal(t, expected, repls.Replace(input))
}

func TestReplacement_Distinct(t *testing.T) {
	rc := ReplacementsContext{Repls: []Replacement{
		{Old: regexp.MustCompile(`\d+`), New: "[NUMBER]", Distinct: true},
	}}

	got := rc.Replace("25\n35\n25")
	assert.Equal(t, "[NUMBER][0]\n[NUMBER][1]\n[NUMBER][0]", got)
}

func TestReplacement_DistinctSingleMatch(t *testing.T) {
	rc := ReplacementsContext{Repls: []Replacement{
		{Old: regexp.MustCompile(`\d+`), New: "[NUMBER]", Distinct: true},
	}}

	got := rc.Replace("25")
	assert.Equal(t, "[NUMBER]", got)
}

func TestReplacement_DistinctSingleValueRepeated(t *testing.T) {
	rc := ReplacementsContext{Repls: []Replacement{
		{Old: regexp.MustCompile(`\d+`), New: "[NUMBER]", Distinct: true},
	}}

	// A single distinct value, even repeated, stays unsuffixed: numbering only
	// adds signal when there is more than one distinct value to tell apart.
	got := rc.Replace("25\n25")
	assert.Equal(t, "[NUMBER]\n[NUMBER]", got)
}

func TestReplaceAppliesInOrder(t *testing.T) {
	input := "A"

	rc := ReplacementsContext{Repls: []Replacement{
		{Old: regexp.MustCompile("B"), New: "C", Order: 2},
		{Old: regexp.MustCompile("A"), New: "B", Order: 1},
	}}

	got := rc.Replace(input)
	assert.Equal(t, "C", got)
}

func TestReplacement_WholeWord(t *testing.T) {
	tests := []struct {
		name  string
		value string
		input string
		want  string
	}{
		{"standalone", "339", "job 339 done", "job [ID] done"},
		{"inside hex string is not replaced", "339", "commit 33939489e027", "commit 33939489e027"},
		{"inside a longer number is not replaced", "339", "id 3390", "id 3390"},
		{"whole text", "339", "339", "[ID]"},
		{"underscore is a separator", "339", "job_339", "job_[ID]"},
		{"dash is a separator", "339", "deploy-339", "deploy-[ID]"},
		{"slash is a separator", "339", "/339/x", "/[ID]/x"},
		{"adjacent matches separated by one character", "339", "339,339", "[ID],[ID]"},
		{"value ending with non-alnum has no check on that side", "/tmp/x/", "/tmp/x/abc /tmp/x/", "[ID]abc [ID]"},
		{"value starting with non-alnum has no check on that side", "-ab", "x-ab", "x[ID]"},
		{"rejected match does not hide a later one", "a.a", "ba.a.a", "ba.[ID]"},
		{"non-ASCII letter is a separator", "339", "é339", "é[ID]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var repls ReplacementsContext
			repls.SetWholeWordWithOrder(tt.value, "[ID]", 0)
			assert.Equal(t, tt.want, repls.Replace(tt.input))
		})
	}
}

func TestReplacement_WholeWordJSON(t *testing.T) {
	var repls ReplacementsContext

	repls.SetWholeWordWithOrder(`a"1`, "[V]", 0)
	// The JSON-encoded form of the value, a\"1, is matched inside JSON output.
	assert.Equal(t, `"[V]"`, repls.Replace(`"a\"1"`))
	assert.Equal(t, `"xa\"1"`, repls.Replace(`"xa\"1"`))
}

func TestReplacement_PlainLiteralUnchanged(t *testing.T) {
	// Values from the harness (Set) keep matching inside longer strings, e.g. paths and versions.
	var repls ReplacementsContext

	repls.Set("339", "[ID]")
	assert.Equal(t, "[ID]39489e027", repls.Replace("33939489e027"))
}
