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

	for _, filter := range []string{"user=", "experiment=", "accelerator_type=", "num_accelerators="} {
		t.Run("empty "+filter, func(t *testing.T) {
			_, err := parseListFilters([]string{filter})
			require.Error(t, err)
			assert.Contains(t, err.Error(), "value cannot be empty")
		})
	}

	t.Run("malformed experiment glob", func(t *testing.T) {
		_, err := parseListFilters([]string{"experiment=["})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid glob pattern")
	})

	for _, acceleratorType := range []string{"1xA10", "8xH100", "GPU_1xA10", "GPU_1xH100", "GPU_8xH100"} {
		t.Run("accelerator type "+acceleratorType, func(t *testing.T) {
			f, err := parseListFilters([]string{"accelerator_type=" + acceleratorType})
			require.NoError(t, err)
			assert.Equal(t, acceleratorType, f.AcceleratorType)
		})
	}
}

func TestResolveAcceleratorFilter(t *testing.T) {
	options := []computeOption{
		{HardwareAccelerator: "GPU_1xA10", DisplayName: "1x A10"},
		{HardwareAccelerator: "GPU_1xH100", DisplayName: "1x H100"},
		{HardwareAccelerator: "GPU_8xH100", DisplayName: "8x H100"},
		{HardwareAccelerator: "GPU_4xA100", DisplayName: "4x A100"},
	}

	cases := []struct {
		name      string
		value     string
		wantTypes []string
		wantCount *int
	}{
		{"family", "H100", []string{"GPU_1xH100", "GPU_8xH100"}, nil},
		{"display name", "8x H100", []string{"GPU_1xH100", "GPU_8xH100"}, new(8)},
		{"compact display name", "8xH100", []string{"GPU_1xH100", "GPU_8xH100"}, new(8)},
		{"full type", "GPU_1xA10", []string{"GPU_1xA10"}, nil},
		{"future family", "A100", []string{"GPU_4xA100"}, nil},
		{"future full type", "GPU_4xA100", []string{"GPU_4xA100"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := listFilters{AcceleratorType: tc.value}
			require.NoError(t, resolveAcceleratorFilter(&f, options))
			assert.Equal(t, tc.wantTypes, f.AcceleratorTypes)
			assert.Equal(t, tc.wantCount, f.AcceleratorCount)
		})
	}

	t.Run("unsupported", func(t *testing.T) {
		f := listFilters{AcceleratorType: "GPU_1xV100"}
		err := resolveAcceleratorFilter(&f, options)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "valid values for this workspace")
		assert.Contains(t, err.Error(), "A10, H100, A100")
		assert.Contains(t, err.Error(), "1x A10, 1x H100, 8x H100, 4x A100")
		assert.Contains(t, err.Error(), "GPU_4xA100")
	})

	t.Run("no available types", func(t *testing.T) {
		f := listFilters{AcceleratorType: "H100"}
		err := resolveAcceleratorFilter(&f, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no accelerator types are available")
	})
}

func TestListFiltersMatches(t *testing.T) {
	run := airRun(1, "me@example.com", "GPU_8xH100", 8, "/Users/me@example.com/qwen-eval")

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
		{"accelerator family", listFilters{AcceleratorType: "h100", AcceleratorTypes: []string{"GPU_1xH100", "GPU_8xH100"}}, true},
		{"accelerator display name", listFilters{AcceleratorType: "8x H100", AcceleratorTypes: []string{"GPU_1xH100", "GPU_8xH100"}, AcceleratorCount: new(8)}, true},
		{"accelerator display count no match", listFilters{AcceleratorType: "1x H100", AcceleratorTypes: []string{"GPU_1xH100", "GPU_8xH100"}, AcceleratorCount: new(1)}, false},
		{"accelerator full type", listFilters{AcceleratorType: "gpu_8xh100", AcceleratorTypes: []string{"GPU_8xH100"}}, true},
		{"accelerator full type other topology", listFilters{AcceleratorType: "GPU_1xH100", AcceleratorTypes: []string{"GPU_1xH100"}}, false},
		{"accelerator type no match", listFilters{AcceleratorType: "a10", AcceleratorTypes: []string{"GPU_1xA10"}}, false},
		{"num accelerators match", listFilters{NumAccelerators: new(8)}, true},
		{"num accelerators no match", listFilters{NumAccelerators: new(4)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.f.matches(&run))
		})
	}
}
