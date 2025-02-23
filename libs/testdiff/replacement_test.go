package testdiff

import (
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
