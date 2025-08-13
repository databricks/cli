package pipelines

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/pipelines"
)

func TestUpdatesBefore(t *testing.T) {
	// Test data: updates sorted in descending order (newest first)
	updates := []pipelines.UpdateInfo{
		{CreationTime: 1000}, // newest
		{CreationTime: 800},
		{CreationTime: 600},
		{CreationTime: 400},
		{CreationTime: 200}, // oldest
	}

	tests := []struct {
		name          string
		timestamp     int64
		expectedCount int
		expectedFirst int64
	}{
		{
			name:          "before 700",
			timestamp:     700,
			expectedCount: 3,
			expectedFirst: 600,
		},
		{
			name:          "before 1000",
			timestamp:     1000,
			expectedCount: 5,
			expectedFirst: 1000,
		},
		{
			name:          "before 200",
			timestamp:     200,
			expectedCount: 1,
			expectedFirst: 200,
		},
		{
			name:          "before 100",
			timestamp:     100,
			expectedCount: 0,
			expectedFirst: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updatesBefore(updates, tt.timestamp)
			if len(result) != tt.expectedCount {
				t.Errorf("updatesBefore(updates, %d) returned %d updates, want %d", tt.timestamp, len(result), tt.expectedCount)
			}
			if tt.expectedCount > 0 && result[0].CreationTime != tt.expectedFirst {
				t.Errorf("updatesBefore(updates, %d) first update = %d, want %d", tt.timestamp, result[0].CreationTime, tt.expectedFirst)
			}
		})
	}
}

func TestUpdatesAfter(t *testing.T) {
	// Test data: updates sorted in descending order (newest first)
	updates := []pipelines.UpdateInfo{
		{CreationTime: 1000}, // newest
		{CreationTime: 800},
		{CreationTime: 600},
		{CreationTime: 400},
		{CreationTime: 200}, // oldest
	}

	tests := []struct {
		name          string
		timestamp     int64
		expectedCount int
		expectedFirst int64
	}{
		{
			name:          "after 500",
			timestamp:     500,
			expectedCount: 3,
			expectedFirst: 1000,
		},
		{
			name:          "after 200",
			timestamp:     200,
			expectedCount: 5,
			expectedFirst: 1000,
		},
		{
			name:          "after 1000",
			timestamp:     1000,
			expectedCount: 1,
			expectedFirst: 1000,
		},
		{
			name:          "after 1200",
			timestamp:     1200,
			expectedCount: 0,
			expectedFirst: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updatesAfter(updates, tt.timestamp)
			if len(result) != tt.expectedCount {
				t.Errorf("updatesAfter(updates, %d) returned %d updates, want %d", tt.timestamp, len(result), tt.expectedCount)
			}
			if tt.expectedCount > 0 && result[0].CreationTime != tt.expectedFirst {
				t.Errorf("updatesAfter(updates, %d) first update = %d, want %d", tt.timestamp, result[0].CreationTime, tt.expectedFirst)
			}
		})
	}
}

func TestUpdatesEmptySlice(t *testing.T) {
	updates := []pipelines.UpdateInfo{}

	tests := []struct {
		name     string
		function func([]pipelines.UpdateInfo, int64) []pipelines.UpdateInfo
	}{
		{
			name:     "updatesBefore",
			function: updatesBefore,
		},
		{
			name:     "updatesAfter",
			function: updatesAfter,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function(updates, 100)
			if result != nil {
				t.Errorf("%s(empty slice) = %v, want nil", tt.name, result)
			}
		})
	}
}
