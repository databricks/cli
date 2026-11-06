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

	PrepareReplacementsUUID(t, &repls, true)

	// A lone UUID stays unsuffixed even in unique mode.
	assert.Equal(t, "[UUID]", repls.Replace("123e4567-e89b-12d3-a456-426614174000"))
}

func TestReplacement_UUIDUnique(t *testing.T) {
	var repls ReplacementsContext

	PrepareReplacementsUUID(t, &repls, true)

	input := "a: 123e4567-e89b-12d3-a456-426614174000\n" +
		"b: 00000000-0000-0000-0000-000000000001\n" +
		"a again: 123e4567-e89b-12d3-a456-426614174000"
	expected := "a: [UUID][0]\n" +
		"b: [UUID][1]\n" +
		"a again: [UUID][0]"
	assert.Equal(t, expected, repls.Replace(input))
}

func TestReplacement_UUIDNotUnique(t *testing.T) {
	var repls ReplacementsContext

	PrepareReplacementsUUID(t, &repls, false)

	input := "a: 123e4567-e89b-12d3-a456-426614174000\n" +
		"b: 00000000-0000-0000-0000-000000000001"
	expected := "a: [UUID]\n" +
		"b: [UUID]"
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

func TestReplaceAppliesInOrder(t *testing.T) {
	input := "A"

	rc := ReplacementsContext{Repls: []Replacement{
		{Old: regexp.MustCompile("B"), New: "C", Order: 2},
		{Old: regexp.MustCompile("A"), New: "B", Order: 1},
	}}

	got := rc.Replace(input)
	assert.Equal(t, "C", got)
}
