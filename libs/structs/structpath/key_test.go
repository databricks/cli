package structpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyStringRoundTrip(t *testing.T) {
	for _, s := range []string{
		"resources.jobs.foo",
		"resources.jobs.daily job",
		`resources.jobs.sample job's"quotes.permissions`,
		"resources.jobs.foo.tasks[0].task_key",
		"a[1][2]",
	} {
		p, err := ParseKey(s)
		require.NoError(t, err, s)
		assert.Equal(t, s, p.KeyString())
	}
	p, err := ParseKey("resources.jobs.daily job")
	require.NoError(t, err)
	assert.Equal(t, "daily job", p.KeyAt(2))
}

func TestParseKeyErrors(t *testing.T) {
	for _, s := range []string{"", "a..b", "a.", "a[x]", "a[1]b"} {
		_, err := ParseKey(s)
		assert.Error(t, err, s)
	}
}
