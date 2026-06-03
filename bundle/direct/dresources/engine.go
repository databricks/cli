package dresources

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/cli/libs/structs/structdiff"
)

// Engine provides state persistence to resource implementations.
// Pass it to DoCreate or DoUpdate to save intermediate state before long-running
// wait operations, so the resource is not orphaned if deployment is interrupted.
type Engine struct {
	resourceKey string
	id          string
	stateType   reflect.Type
	saveFunc    func(id string, x any) error
	lastSaved   any
}

// NewEngine creates an Engine with the given state type and save function.
// The framework calls this before invoking DoCreate or DoUpdate.
func NewEngine(resourceKey string, stateType reflect.Type, saveFunc func(id string, x any) error) *Engine {
	return &Engine{resourceKey: resourceKey, id: "", stateType: stateType, saveFunc: saveFunc, lastSaved: nil}
}

// NewNopEngine creates an Engine that discards all saves. Use in tests.
func NewNopEngine(stateType reflect.Type) *Engine {
	return NewEngine("", stateType, func(_ string, _ any) error { return nil })
}

// SaveState saves the resource state. id must be the resource's identifier; on
// the first call it is recorded, and subsequent calls panic if a different id is
// passed. x must be a pointer to the same struct type as the resource's state.
// If the state is identical to what was last saved, the write is skipped.
// Failures to persist state are logged but do not abort the deployment — the
// resource already exists and aborting would not undo its creation.
func (e *Engine) SaveState(ctx context.Context, id string, x any) {
	if e.id == "" {
		e.id = id
	} else if e.id != id {
		panic(fmt.Sprintf("SaveState: id mismatch: expected %q, got %q", e.id, id))
	}
	xt := reflect.TypeOf(x)
	if xt != e.stateType {
		panic(fmt.Sprintf("SaveState: type mismatch: expected %v, got %v", e.stateType, xt))
	}
	if e.lastSaved != nil && structdiff.IsEqual(e.lastSaved, x) {
		log.Debugf(ctx, "SaveState: %s id=%s: skipping, state unchanged", e.resourceKey, id)
		return
	}
	b, _ := json.Marshal(x)
	preview := string(b)
	if len(preview) > 100 {
		preview = preview[:100]
	}
	log.Debugf(ctx, "SaveState: %s id=%s %d bytes: %s", e.resourceKey, id, len(b), preview)
	if err := e.saveFunc(e.id, x); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	e.lastSaved = x
}
