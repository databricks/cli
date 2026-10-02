package root

import (
	"testing"
	"time"

	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/dbr"
	"github.com/stretchr/testify/assert"
)

func TestStaleVersionWarning(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	stale := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	header := "Warning: this version of the Databricks CLI was built on 2026-01-15 and is more than 6 months old. " +
		"We strongly recommend updating to the latest version.\n"

	tests := []struct {
		name      string
		buildTime time.Time
		command   string
		want      string
	}{
		{name: "no embedded timestamp", buildTime: time.Unix(0, 0), command: "brew upgrade databricks"},
		{name: "recent build", buildTime: now.AddDate(0, -1, 0), command: "brew upgrade databricks"},
		{name: "just under threshold", buildTime: now.Add(-staleVersionThreshold + time.Hour)},
		{
			name:      "stale with detected install method",
			buildTime: stale,
			command:   "brew upgrade databricks",
			want:      header + "To upgrade, run: brew upgrade databricks\n",
		},
		{
			name:      "stale with unknown install method",
			buildTime: stale,
			want:      header + "See " + installDocsURL + " to upgrade.\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, staleVersionWarning(tc.buildTime, now, tc.command))
		})
	}
}

func TestSkipStaleVersionWarning(t *testing.T) {
	release := build.Info{Version: "1.18.0"}
	notDBR := dbr.Environment{}

	tests := []struct {
		name    string
		info    build.Info
		runtime dbr.Environment
		want    bool
	}{
		{name: "release build", info: release, runtime: notDBR},
		{name: "dev build", info: build.Info{Version: "1.19.0-dev+abc"}, runtime: notDBR, want: true},
		{name: "snapshot build", info: build.Info{Version: "1.18.0", IsSnapshot: true}, runtime: notDBR, want: true},
		{name: "on DBR", info: release, runtime: dbr.Environment{IsDbr: true, Version: "15.4"}, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := dbr.MockRuntime(t.Context(), tc.runtime)
			assert.Equal(t, tc.want, skipStaleVersionWarning(ctx, tc.info))
		})
	}
}
