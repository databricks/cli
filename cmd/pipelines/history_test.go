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
		timestamp     *int64
		expectedCount int
		expectedFirst int64
	}{
		{
			name:          "before 700",
			timestamp:     int64Ptr(700),
			expectedCount: 3,
			expectedFirst: 600,
		},
		{
			name:          "before 1000",
			timestamp:     int64Ptr(1000),
			expectedCount: 5,
			expectedFirst: 1000,
		},
		{
			name:          "before 200",
			timestamp:     int64Ptr(200),
			expectedCount: 1,
			expectedFirst: 200,
		},
		{
			name:          "before 100",
			timestamp:     int64Ptr(100),
			expectedCount: 0,
			expectedFirst: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updatesBefore(updates, *tt.timestamp)
			if len(result) != tt.expectedCount {
				t.Errorf("updatesBefore(updates, %d) returned %d updates, want %d", *tt.timestamp, len(result), tt.expectedCount)
			}
			if tt.expectedCount > 0 && result[0].CreationTime != tt.expectedFirst {
				t.Errorf("updatesBefore(updates, %d) first update = %d, want %d", *tt.timestamp, result[0].CreationTime, tt.expectedFirst)
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
		timestamp     *int64
		expectedCount int
		expectedFirst int64
	}{
		{
			name:          "after 500",
			timestamp:     int64Ptr(500),
			expectedCount: 3,
			expectedFirst: 1000,
		},
		{
			name:          "after 200",
			timestamp:     int64Ptr(200),
			expectedCount: 5,
			expectedFirst: 1000,
		},
		{
			name:          "after 1000",
			timestamp:     int64Ptr(1000),
			expectedCount: 1,
			expectedFirst: 1000,
		},
		{
			name:          "after 1200",
			timestamp:     int64Ptr(1200),
			expectedCount: 0,
			expectedFirst: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := updatesAfter(updates, *tt.timestamp)
			if len(result) != tt.expectedCount {
				t.Errorf("updatesAfter(updates, %d) returned %d updates, want %d", *tt.timestamp, len(result), tt.expectedCount)
			}
			if tt.expectedCount > 0 && result[0].CreationTime != tt.expectedFirst {
				t.Errorf("updatesAfter(updates, %d) first update = %d, want %d", *tt.timestamp, result[0].CreationTime, tt.expectedFirst)
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

	var noUpdates []pipelines.UpdateInfo

	oneUpdate := []pipelines.UpdateInfo{
		{CreationTime: 500},
	}

	tests := []struct {
		name          string
		updates       []pipelines.UpdateInfo
		startTime     *int64
		endTime       *int64
		expectedCount int
		expectedFirst int64
		expectedLast  int64
	}{
		{
			name:          "both times nil - return all",
			updates:       updates,
			startTime:     nil,
			endTime:       nil,
			expectedCount: 5,
			expectedFirst: 1000,
			expectedLast:  200,
		},
		{
			name:          "start time nil, end time set",
			updates:       updates,
			startTime:     nil,
			endTime:       int64Ptr(700),
			expectedCount: 3,
			expectedFirst: 600,
			expectedLast:  200,
		},
		{
			name:          "start time set, end time nil",
			updates:       updates,
			startTime:     int64Ptr(500),
			endTime:       nil,
			expectedCount: 3,
			expectedFirst: 1000,
			expectedLast:  600,
		},
		{
			name:          "both times set within range",
			updates:       updates,
			startTime:     int64Ptr(300),
			endTime:       int64Ptr(900),
			expectedCount: 3,
			expectedFirst: 800,
			expectedLast:  400,
		},
		{
			name:          "both times set, no overlap",
			updates:       updates,
			startTime:     int64Ptr(1200),
			endTime:       int64Ptr(1500),
			expectedCount: 0,
			expectedFirst: 0,
			expectedLast:  0,
		},
		{
			name:          "start time after all updates",
			updates:       updates,
			startTime:     int64Ptr(1200),
			endTime:       nil,
			expectedCount: 0,
			expectedFirst: 0,
			expectedLast:  0,
		},
		{
			name:          "end time before all updates",
			updates:       updates,
			startTime:     nil,
			endTime:       int64Ptr(100),
			expectedCount: 0,
			expectedFirst: 0,
			expectedLast:  0,
		},
		{
			name:          "start time after end time but within range",
			updates:       updates,
			startTime:     int64Ptr(700),
			endTime:       int64Ptr(500),
			expectedCount: 0,
			expectedFirst: 0,
			expectedLast:  0,
		},
		{
			name:          "start and end time match exact values in list",
			updates:       updates,
			startTime:     int64Ptr(400),
			endTime:       int64Ptr(800),
			expectedCount: 3,
			expectedFirst: 800,
			expectedLast:  400,
		},
		{
			name:          "no updates - empty slice",
			updates:       noUpdates,
			startTime:     nil,
			endTime:       nil,
			expectedCount: 0,
			expectedFirst: 0,
			expectedLast:  0,
		},
		{
			name:          "one update - both times nil",
			updates:       oneUpdate,
			startTime:     nil,
			endTime:       nil,
			expectedCount: 1,
			expectedFirst: 500,
			expectedLast:  500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := filterUpdates(tt.updates, tt.startTime, tt.endTime)
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

// Helper function to create int64 pointers
func int64Ptr(v int64) *int64 { return &v }
