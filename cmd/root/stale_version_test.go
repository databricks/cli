package root

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/dbr"
	"github.com/databricks/cli/libs/flags"
	"github.com/stretchr/testify/assert"
)

func TestStaleVersionWarning(t *testing.T) {
	now := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)

	// Dev build: no embedded timestamp.
	assert.Empty(t, staleVersionWarning(time.Unix(0, 0), now))
	// Recent build.
	assert.Empty(t, staleVersionWarning(now.AddDate(0, -1, 0), now))
	// Just under the threshold.
	assert.Empty(t, staleVersionWarning(now.Add(-staleVersionThreshold+time.Hour), now))

	// Past the threshold: warns and shows the build date.
	msg := staleVersionWarning(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), now)
	assert.Contains(t, msg, "built on 2026-01-15")
	assert.Contains(t, msg, "strongly recommend updating")
}

func TestWarnIfStaleVersionSkippedOnDBR(t *testing.T) {
	var stderr bytes.Buffer
	ctx := cmdio.MockDiscard(t.Context())
	ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputText, nil, io.Discard, &stderr, "", ""))
	ctx = dbr.MockRuntime(ctx, dbr.Environment{IsDbr: true, Version: "15.4"})

	warnIfStaleVersion(ctx)
	assert.Empty(t, stderr.String())
}
