package acceptance_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/stretchr/testify/assert"
)

// recordingT records log output and whether a failure was reported, so we can
// drive doComparison without failing the enclosing test.
type recordingT struct {
	testutil.TestingT
	ctx    context.Context
	logs   strings.Builder
	failed bool
}

func (r *recordingT) Helper() {}

func (r *recordingT) Log(args ...any) { fmt.Fprintln(&r.logs, args...) }

func (r *recordingT) Logf(format string, args ...any) {
	fmt.Fprintf(&r.logs, format, args...)
}

func (r *recordingT) Error(args ...any) { r.failed = true }

func (r *recordingT) Errorf(format string, args ...any) { r.failed = true }

func (r *recordingT) Context() context.Context { return r.ctx }

// TestSoftFailComparison exercises the soft-fail branch in doComparison directly.
// A committed golden mismatch can't drive an acceptance selftest for this: -update
// would heal it, so the SOFTFAIL path would stop firing. Driving the comparison
// with a recording T lets us assert both the green-on-listed-file path and that an
// unlisted file (including output.txt) still fails.
func TestSoftFailComparison(t *testing.T) {
	tests := []struct {
		name          string
		relPath       string
		softFailFiles []string
		softFailAll   bool
		wantFailed    bool
		wantSoftfail  bool
	}{
		{
			name:          "listed file drifts green with marker",
			relPath:       "out.drifty.txt",
			softFailFiles: []string{"out.drifty.txt"},
			wantFailed:    false,
			wantSoftfail:  true,
		},
		{
			name:          "unlisted file drift still fails",
			relPath:       "out.other.txt",
			softFailFiles: []string{"out.drifty.txt"},
			wantFailed:    true,
			wantSoftfail:  false,
		},
		{
			name:          "output.txt drift still fails",
			relPath:       "output.txt",
			softFailFiles: []string{"out.drifty.txt"},
			wantFailed:    true,
			wantSoftfail:  false,
		},
		{
			name:         "whole-test shield drifts green with marker",
			relPath:      "out.other.txt",
			softFailAll:  true,
			wantFailed:   false,
			wantSoftfail: true,
		},
		{
			name:         "whole-test shield covers output.txt",
			relPath:      "output.txt",
			softFailAll:  true,
			wantFailed:   false,
			wantSoftfail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dirRef := t.TempDir()
			dirNew := t.TempDir()
			testutil.WriteFile(t, filepath.Join(dirRef, tt.relPath), "reference\n")
			testutil.WriteFile(t, filepath.Join(dirNew, tt.relPath), "drifted\n")

			rt := &recordingT{ctx: t.Context()}
			doComparison(rt, testdiff.ReplacementsContext{}, dirRef, dirNew, tt.relPath, tt.softFailFiles, tt.softFailAll, nil)

			assert.Equal(t, tt.wantFailed, rt.failed)
			gotSoftfail := strings.Contains(rt.logs.String(), "SOFTFAIL "+tt.relPath)
			assert.Equal(t, tt.wantSoftfail, gotSoftfail)
		})
	}
}
