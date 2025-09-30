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

	assert.Equal(t, "[UUID]", repls.Replace("123e4567-e89b-12d3-a456-426614174000"))
}

func TestReplacement_Number(t *testing.T) {
	var repls ReplacementsContext

	PrepareReplacementsNumber(t, &repls)

	assert.Equal(t, "12", repls.Replace("12"))
	assert.Equal(t, "[NUMID]", repls.Replace("123"))
}

func TestReplacement_TemporaryDirectory(t *testing.T) {
	var repls ReplacementsContext

	PrepareReplacementsTemporaryDirectory(t, &repls)

	assert.Equal(t, "/tmp/.../tail", repls.Replace("/tmp/foo/bar/qux/tail"))
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
