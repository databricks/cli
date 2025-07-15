package logdiag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
)

type keyType int

const key = keyType(0)

type LogDiagData struct {
	mu *sync.Mutex

	// How many diagnostics if each type were seen
	Errors          int
	Warnings        int
	Recommendations int

	// Only print diagnostics with this severity or lower (default: everything)
	TargetSeverity diag.Severity

	// Root to resolve location against
	Root string

	// If Collect is true, diagnostics are appended to Collected. Use SetCollected() to set.
	Collect   bool
	Collected []diag.Diagnostic
}

// IsSetup returns whether InitContext() was already called.
// It is meant to be used in test code. In production, code InitContext() should be placed
// in unambiguous place instead.
func IsSetup(ctx context.Context) bool {
	_, ok := ctx.Value(key).(*LogDiagData)
	return ok
}

func InitContext(ctx context.Context) context.Context {
	_, ok := ctx.Value(key).(*LogDiagData)
	if ok {
		panic("internal error: must not call InitContext() twice")
	}
	val := LogDiagData{
		TargetSeverity: 255,
		mu:             &sync.Mutex{},
	}
	return context.WithValue(ctx, key, &val)
}

func read(ctx context.Context) *LogDiagData {
	val, ok := ctx.Value(key).(*LogDiagData)
	if !ok || val == nil {
		panic("internal error: must call InitContext() first")
	}
	return val
}

func Copy(ctx context.Context) LogDiagData {
	val := read(ctx)
	val.mu.Lock()
	defer val.mu.Unlock()
	return *val
}

func HasError(ctx context.Context) bool {
	val := read(ctx)
	val.mu.Lock()
	defer val.mu.Unlock()

	return val.Errors > 0
}

func SetSeverity(ctx context.Context, target diag.Severity) {
	val := read(ctx)
	val.mu.Lock()
	defer val.mu.Unlock()

	read(ctx).TargetSeverity = target
}

func SetRoot(ctx context.Context, root string) {
	val := read(ctx)
	val.mu.Lock()
	defer val.mu.Unlock()

	read(ctx).Root = root
}

func SetCollect(ctx context.Context, collect bool) {
	val := read(ctx)
	val.mu.Lock()
	defer val.mu.Unlock()

	read(ctx).Collect = collect
}

func FlushCollected(ctx context.Context) diag.Diagnostics {
	val := read(ctx)
	val.mu.Lock()
	defer val.mu.Unlock()

	result := val.Collected
	val.Collected = nil
	return result
}

func LogDiag(ctx context.Context, d diag.Diagnostic) {
	val := read(ctx)
	val.mu.Lock()
	defer val.mu.Unlock()

	switch d.Severity {
	case diag.Error:
		val.Errors += 1
	case diag.Warning:
		val.Warnings += 1
	case diag.Recommendation:
		val.Recommendations += 1
	}

	if d.Severity > val.TargetSeverity {
		return
	}

	for i := range d.Locations {
		// Make location relative to the root
		if d.Locations[i].File != "" {
			out, err := filepath.Rel(val.Root, d.Locations[i].File)
			if err == nil {
				d.Locations[i].File = out
			}
			// Fix slashes, because:
			// - The forward slashes work on Windows as well.
			// - It's informational.
			// - It avoids output different between OSes which makes it easier to write cross-platform tests.
			d.Locations[i].File = filepath.ToSlash(d.Locations[i].File)
		}
	}

	if val.Collect {
		val.Collected = append(val.Collected, d)
	} else {
		err := cmdio.RenderDiagnostics(os.Stderr, []diag.Diagnostic{d})
		if err != nil {
			fmt.Fprint(os.Stderr, "\nRendering error: "+err.Error()+"\n")
		}
	}
}

func LogError(ctx context.Context, err error) {
	diags := diag.FromErr(err)
	if len(diags) > 0 {
		LogDiag(ctx, diags[0])
	}
}
