package aircmd

import (
	"fmt"
	"strings"
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

	t.Run("unknown key with empty value", func(t *testing.T) {
		_, err := parseListFilters([]string{"acclerator_type="})
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
			key, _, _ := strings.Cut(filter, "=")
			require.EqualError(t, err, fmt.Sprintf("invalid --filter %s=%q: value cannot be empty", key, ""))
		})
	}

	t.Run("trim values", func(t *testing.T) {
		f, err := parseListFilters([]string{"user= me@example.com ", "experiment= qwen* ", "num_accelerators= 8 "})
		require.NoError(t, err)
		assert.Equal(t, "me@example.com", f.User)
		assert.Equal(t, "qwen*", f.Experiment)
		require.NotNil(t, f.NumAccelerators)
		assert.Equal(t, 8, *f.NumAccelerators)
	})

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

func TestValidateAcceleratorFilter(t *testing.T) {
	options := []computeOption{
		{HardwareAccelerator: "GPU_1xA10", DisplayName: "1x A10"},
		{HardwareAccelerator: "GPU_1xH100", DisplayName: "1x H100"},
		{HardwareAccelerator: "GPU_8xH100", DisplayName: "8x H100"},
		{HardwareAccelerator: "GPU_8xB300", DisplayName: "8x B300"},
	}

	for _, value := range []string{"H100", "GPU_1xA10", "B300"} {
		t.Run("valid "+value, func(t *testing.T) {
			require.NoError(t, validateAcceleratorFilter(value, options))
		})
	}

	for value, fullType := range map[string]string{
		"1xA10":   "GPU_1xA10",
		"1x A10":  "GPU_1xA10",
		"1xH100":  "GPU_1xH100",
		"1x H100": "GPU_1xH100",
		"8xH100":  "GPU_8xH100",
		"8x H100": "GPU_8xH100",
		"8x B300": "GPU_8xB300",
	} {
		t.Run("suggest full type for "+value, func(t *testing.T) {
			err := validateAcceleratorFilter(value, options)
			require.EqualError(t, err, fmt.Sprintf("invalid --filter accelerator_type=%q: use %q", value, fullType))
		})
	}

	t.Run("unsupported", func(t *testing.T) {
		err := validateAcceleratorFilter("GPU_1xV100", options)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "valid values for this workspace")
		assert.Contains(t, err.Error(), "A10, H100, B300")
		assert.Contains(t, err.Error(), "GPU_8xB300")
	})

	t.Run("no available types", func(t *testing.T) {
		err := validateAcceleratorFilter("H100", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no accelerator types are available")
	})
}

func TestLegacyAcceleratorSubstring(t *testing.T) {
	f := listFilters{AcceleratorType: "H1"}
	configureAcceleratorFilter(&f)
	assert.True(t, f.matchesFields(filterFields{GPUType: "h100_80gb"}))
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
		{"accelerator family", listFilters{AcceleratorType: "h100", AcceleratorFamilyMatch: true}, true},
		{"accelerator full type", listFilters{AcceleratorType: "gpu_8xh100", ResolvedAcceleratorType: "GPU_8xH100"}, true},
		{"accelerator full type other topology", listFilters{AcceleratorType: "GPU_1xH100", ResolvedAcceleratorType: "GPU_1xH100"}, false},
		{"accelerator type no match", listFilters{AcceleratorType: "GPU_1xA10", ResolvedAcceleratorType: "GPU_1xA10"}, false},
		{"num accelerators match", listFilters{NumAccelerators: new(8)}, true},
		{"num accelerators no match", listFilters{NumAccelerators: new(4)}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, c.f.matches(&run))
		})
	}

	t.Run("canonical type uses derived family", func(t *testing.T) {
		fields := filterFields{GPUType: "GPU_8xB300"}
		assert.True(t, (listFilters{AcceleratorType: "B300", AcceleratorFamilyMatch: true}).matchesFields(fields))
		assert.False(t, (listFilters{AcceleratorType: "8xB300", AcceleratorFamilyMatch: true}).matchesFields(fields))
	})
}
