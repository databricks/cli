package phases

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetExecutionTimes(t *testing.T) {
	tests := []struct {
		name                string
		inputExecutionTimes []protos.IntMapEntry
		expectedKeys        []string
		expectedValues      []int64
		description         string
	}{
		{
			name: "mix of above and below 1ms",
			inputExecutionTimes: []protos.IntMapEntry{
				{Key: "mutator1", Value: 3000}, // 3ms
				{Key: "mutator2", Value: 2000}, // 2ms
				{Key: "mutator3", Value: 800},  // 0.8ms
				{Key: "mutator4", Value: 600},  // 0.6ms
				{Key: "mutator5", Value: 1500}, // 1.5ms
			},
			expectedKeys:   []string{"mutator1", "mutator2", "mutator5", "mutator3", "mutator4"},
			expectedValues: []int64{3, 2, 1, 0, 0},
			description:    "should keep all mutators (sorted by execution time)",
		},
		{
			name: "all mutators above 1ms",
			inputExecutionTimes: []protos.IntMapEntry{
				{Key: "mutator1", Value: 5000}, // 5ms
				{Key: "mutator2", Value: 3000}, // 3ms
				{Key: "mutator3", Value: 2000}, // 2ms
				{Key: "mutator4", Value: 1500}, // 1.5ms
				{Key: "mutator5", Value: 1200}, // 1.2ms
				{Key: "mutator6", Value: 1100}, // 1.1ms
			},
			expectedKeys:   []string{"mutator1", "mutator2", "mutator3", "mutator4", "mutator5", "mutator6"},
			expectedValues: []int64{5, 3, 2, 1, 1, 1},
			description:    "should keep all mutators above 1ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a bundle with the test execution times
			b := &bundle.Bundle{
				Metrics: bundle.Metrics{
					ExecutionTimes: tt.inputExecutionTimes,
				},
			}

			// Call the function
			result := getExecutionTimes(b)

			// Verify the results
			require.Len(t, result, len(tt.expectedKeys), tt.description)

			for i, expectedKey := range tt.expectedKeys {
				assert.Equal(t, expectedKey, result[i].Key, "Key mismatch at index %d: %s", i, tt.description)
				assert.Equal(t, tt.expectedValues[i], result[i].Value, "Value mismatch at index %d: %s", i, tt.description)
			}
		})
	}
}

func TestGetExecutionTimes_250Limit(t *testing.T) {
	// Test the 250 entry limit
	var executionTimes []protos.IntMapEntry
	for i := range 300 {
		executionTimes = append(executionTimes, protos.IntMapEntry{
			Key:   fmt.Sprintf("mutator%d", i),
			Value: int64(1000 + i), // All above 1ms to avoid sub-millisecond filtering
		})
	}

	b := &bundle.Bundle{
		Metrics: bundle.Metrics{
			ExecutionTimes: executionTimes,
		},
	}

	result := getExecutionTimes(b)

	// Should be limited to 250 entries
	assert.Len(t, result, 250, "should limit to 250 entries")

	// Should be the top 250 (highest values due to descending sort)
	assert.Equal(t, "mutator299", result[0].Key, "first entry should be highest value")
	assert.Equal(t, "mutator50", result[249].Key, "last entry should be 250th highest value")
}
