package aircmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseListFilters(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		f, err := parseListFilters([]string{
			"user=me@example.com",
			"experiment=qwen*",
			"accelerator_type=H100",
			"num_accelerators=8",
		})
		require.NoError(t, err)
		assert.Equal(t, "me@example.com", f.User)
		assert.Equal(t, "qwen*", f.Experiment)
		assert.Equal(t, "H100", f.AcceleratorType)
		require.NotNil(t, f.NumAccelerators)
		assert.Equal(t, 8, *f.NumAccelerators)
	})

	t.Run("unknown key", func(t *testing.T) {
		_, err := parseListFilters([]string{"region=us"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported --filter key")
	})

	t.Run("malformed pair", func(t *testing.T) {
		_, err := parseListFilters([]string{"experiment"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "expected KEY=VALUE")
	})

	t.Run("bad num_accelerators", func(t *testing.T) {
		_, err := parseListFilters([]string{"num_accelerators=lots"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "num_accelerators")
	})

	t.Run("non-positive num_accelerators", func(t *testing.T) {
		_, err := parseListFilters([]string{"num_accelerators=0"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "positive integer")
	})
}

func TestListFiltersMatches(t *testing.T) {
	run := airJobRun(1, "me@example.com", "GPU_8xH100", 8, "/Users/me@example.com/qwen-eval")

	cases := []struct {
		name string
		f    listFilters
		want bool
	}{
		{"no filters", listFilters{}, true},
		{"experiment prefix glob", listFilters{Experiment: "qwen*"}, true},
		{"experiment suffix glob", listFilters{Experiment: "*-eval"}, true},
		{"experiment case-insensitive", listFilters{Experiment: "QWEN*"}, true},
		{"experiment no match", listFilters{Experiment: "llama*"}, false},
		{"accelerator type substring", listFilters{AcceleratorType: "h100"}, true},
		{"accelerator type no match", listFilters{AcceleratorType: "a10"}, false},
		{"num accelerators match", listFilters{NumAccelerators: new(8)}, true},
		{"num accelerators no match", listFilters{NumAccelerators: new(4)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.f.matches(&run))
		})
	}
}
