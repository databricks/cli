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

const key = keyType(2)

var Mu sync.Mutex

type Value struct {
	Errors          int
	Warnings        int
	Recommendations int
	Root            string

	Collect   bool
	Collected []diag.Diagnostic
}

// IsSetup is test helper, do not use in production code
func IsSetup(ctx context.Context) bool {
	_, ok := ctx.Value(key).(*Value)
	return ok
}

func InitContext(ctx context.Context) context.Context {
	_, ok := ctx.Value(key).(*Value)
	if ok {
		panic("internal error: must not call InitContext() twice")
	}
	val := Value{}
	return context.WithValue(ctx, key, &val)
}

func SetRoot(ctx context.Context, root string) {
	read(ctx).Root = root
}

func read(ctx context.Context) *Value {
	val, ok := ctx.Value(key).(*Value)
	if !ok || val == nil {
		panic("internal error: must call InitContext() first")
	}
	return val
}

func GetContext(ctx context.Context) Value {
	Mu.Lock()
	defer Mu.Unlock()
	return *read(ctx)
}

func HasError(ctx context.Context) bool {
	Mu.Lock()
	defer Mu.Unlock()
	return read(ctx).Errors > 0
}

func SetCollect(ctx context.Context, collect bool) {
	Mu.Lock()
	defer Mu.Unlock()
	read(ctx).Collect = collect
}

func GetCollected(ctx context.Context) []diag.Diagnostic {
	Mu.Lock()
	defer Mu.Unlock()
	return read(ctx).Collected
}

func FlushCollected(ctx context.Context) []diag.Diagnostic {
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

	for i := range d.Locations {
		// Make location relative to the root
		if d.Locations[i].File != "" {
			out, err := filepath.Rel(val.Root, d.Locations[i].File)
			if err == nil {
				d.Locations[i].File = out
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
		// Convert error to diagnostic and collect it
		diags := diag.FromErr(err)
		val.Collected = append(val.Collected, diags...)
	} else {
		cmdio.LogError(ctx, err)
	}
}
