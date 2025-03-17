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

	text := `123e4567-e89b-12d3-a456-426614174000 123e4567-e89b-12d3-a456-426614174000
481574F3-C361-4347-B45A-DB9D8FC86D21
2370DD61-95E2-44A8-8BC5-CA28F2C303F8`

	assert.Equal(t, `[UUID-0] [UUID-0]
[UUID-1]
[UUID-2]`, repls.Replace(text))

	// State from a previous replacement should be retained.
	text2 := `481574F3-C361-4347-B45A-DB9D8FC86D21 6FC87703-D9BD-4DFD-A7A7-88C423F3A124`
	assert.Equal(t, `[UUID-1] [UUID-3]`, repls.Replace(text2))
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
