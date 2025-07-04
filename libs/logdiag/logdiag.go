package logdiag

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/diag"
)

type keyType int

const key = keyType(0)

// Global mutex is used because we write to stderr.
var Mu sync.Mutex

type LogDiagData struct {
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

// IsSetup is test helper, do not use in production code
func IsSetup(ctx context.Context) bool {
	Mu.Lock()
	defer Mu.Unlock()

	_, ok := ctx.Value(key).(*LogDiagData)
	return ok
}

func InitContext(ctx context.Context) context.Context {
	Mu.Lock()
	defer Mu.Unlock()

	_, ok := ctx.Value(key).(*LogDiagData)
	if ok {
		panic("internal error: must not call InitContext() twice")
	}
	val := LogDiagData{
		TargetSeverity: 255,
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
	Mu.Lock()
	defer Mu.Unlock()

	return *read(ctx)
}

func HasError(ctx context.Context) bool {
	Mu.Lock()
	defer Mu.Unlock()

	return read(ctx).Errors > 0
}

func SetSeverity(ctx context.Context, target diag.Severity) {
	Mu.Lock()
	defer Mu.Unlock()

	read(ctx).TargetSeverity = target
}

func SetRoot(ctx context.Context, root string) {
	Mu.Lock()
	defer Mu.Unlock()

	read(ctx).Root = root
}

func SetCollect(ctx context.Context, collect bool) {
	Mu.Lock()
	defer Mu.Unlock()

	read(ctx).Collect = collect
}

func FlushCollected(ctx context.Context) diag.Diagnostics {
	Mu.Lock()
	defer Mu.Unlock()

	val := read(ctx)
	result := val.Collected
	val.Collected = nil
	return result
}

func LogDiag(ctx context.Context, d diag.Diagnostic) {
	Mu.Lock()
	defer Mu.Unlock()

	val := read(ctx)
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
			if runtime.GOOS == "windows" {
				d.Locations[i].File = strings.ReplaceAll(d.Locations[i].File, "\\", "/")
			}
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
	Mu.Lock()
	defer Mu.Unlock()

	val := read(ctx)
	val.Errors += 1

	if val.Collect {
		diags := diag.FromErr(err)
		val.Collected = append(val.Collected, diags...)
	} else {
		cmdio.LogError(ctx, err)
	}
}
