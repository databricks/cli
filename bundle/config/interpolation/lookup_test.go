package interpolation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type interpolationFixture struct {
	A map[string]string `json:"a"`
	B map[string]string `json:"b"`
	C map[string]string `json:"c"`
}

func fixture() interpolationFixture {
	return interpolationFixture{
		A: map[string]string{
			"x": "1",
		},
		B: map[string]string{
			"x": "2",
		},
		C: map[string]string{
			"ax": "${a.x}",
			"bx": "${b.x}",
		},
	}
}

func TestExcludePath(t *testing.T) {
	tmp := fixture()
	m := interpolate{
		fn: ExcludeLookupsInPath("a"),
	}

	err := m.expand(&tmp)
	require.NoError(t, err)

	assert.Equal(t, "1", tmp.A["x"])
	assert.Equal(t, "2", tmp.B["x"])
	assert.Equal(t, "${a.x}", tmp.C["ax"])
	assert.Equal(t, "2", tmp.C["bx"])
}

func TestIncludePath(t *testing.T) {
	tmp := fixture()
	m := interpolate{
		fn: IncludeLookupsInPath("a"),
	}

	err := m.expand(&tmp)
	require.NoError(t, err)

	assert.Equal(t, "1", tmp.A["x"])
	assert.Equal(t, "2", tmp.B["x"])
	assert.Equal(t, "1", tmp.C["ax"])
	assert.Equal(t, "${b.x}", tmp.C["bx"])
}
