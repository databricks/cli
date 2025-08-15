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
	var updates []pipelines.UpdateInfo

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
			if len(result) != 0 {
				t.Errorf("%s(empty slice) returned %d items, want empty array", tt.name, len(result))
			}
		})
	}
}

func TestFilterUpdates(t *testing.T) {
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
		startTime     int64
		endTime       int64
		expectedCount int
		expectedFirst int64
		expectedLast  int64
	}{
		{
			name:          "both times zero - return all",
			startTime:     0,
			endTime:       0,
			expectedCount: 5,
			expectedFirst: 1000,
			expectedLast:  200,
		},
		{
			name:          "start time zero, end time set",
			startTime:     0,
			endTime:       700,
			expectedCount: 3,
			expectedFirst: 600,
			expectedLast:  200,
		},
		{
			name:          "start time set, end time zero",
			startTime:     500,
			endTime:       0,
			expectedCount: 3,
			expectedFirst: 1000,
			expectedLast:  600,
		},
		{
			name:          "both times set within range",
			startTime:     300,
			endTime:       900,
			expectedCount: 3,
			expectedFirst: 800,
			expectedLast:  400,
		},
		{
			name:          "both times set, no overlap",
			startTime:     1200,
			endTime:       1500,
			expectedCount: 0,
			expectedFirst: 0,
			expectedLast:  0,
		},
		{
			name:          "start time after all updates",
			startTime:     1200,
			endTime:       0,
			expectedCount: 0,
			expectedFirst: 0,
			expectedLast:  0,
		},
		{
			name:          "end time before all updates",
			startTime:     0,
			endTime:       100,
			expectedCount: 0,
			expectedFirst: 0,
			expectedLast:  0,
		},
		{
			name:          "start time after end time but within range",
			startTime:     700,
			endTime:       500,
			expectedCount: 0,
			expectedFirst: 0,
			expectedLast:  0,
		},
		{
			name:          "start and end time match exact values in list",
			startTime:     400,
			endTime:       800,
			expectedCount: 3,
			expectedFirst: 800,
			expectedLast:  400,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filterUpdates(updates, tt.startTime, tt.endTime)
			if err != nil {
				t.Errorf("filterUpdates(updates, %d, %d) returned error: %v", tt.startTime, tt.endTime, err)
			}
			if len(result) != tt.expectedCount {
				t.Errorf("filterUpdates(updates, %d, %d) returned %d updates, want %d", tt.startTime, tt.endTime, len(result), tt.expectedCount)
			}
			if tt.expectedCount > 0 {
				if result[0].CreationTime != tt.expectedFirst {
					t.Errorf("filterUpdates(updates, %d, %d) first update = %d, want %d", tt.startTime, tt.endTime, result[0].CreationTime, tt.expectedFirst)
				}
				if result[len(result)-1].CreationTime != tt.expectedLast {
					t.Errorf("filterUpdates(updates, %d, %d) last update = %d, want %d", tt.startTime, tt.endTime, result[len(result)-1].CreationTime, tt.expectedLast)
				}
			}
		})
	}
}
