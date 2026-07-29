package dresources

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// StateSaver provides state persistence to resource implementations.
// Pass it to DoCreate or DoUpdate to save intermediate state before long-running
// wait operations, so the resource is not orphaned if deployment is interrupted.
type StateSaver struct {
	resourceKey string
	id          string
	stateType   reflect.Type
	saveFunc    func(id string, b json.RawMessage) error
	lastSaved   []byte // JSON snapshot; stored by value to avoid aliasing with the live config pointer
}

// NewStateSaver creates an StateSaver with the given state type and save function.
// The framework calls this before invoking DoCreate or DoUpdate.
func NewStateSaver(resourceKey string, stateType reflect.Type, saveFunc func(id string, b json.RawMessage) error) *StateSaver {
	return &StateSaver{resourceKey: resourceKey, id: "", stateType: stateType, saveFunc: saveFunc, lastSaved: nil}
}

// NewNopStateSaver creates an StateSaver that discards all saves. Use in tests.
func NewNopStateSaver(stateType reflect.Type) *StateSaver {
	return NewStateSaver("", stateType, func(_ string, _ json.RawMessage) error { return nil })
}

// SaveStateWith saves state with field temporarily set to value, then restores it.
// This is useful when the actual current state of a field differs from its desired
// value in config — e.g. saving started=true before a stop, or published=false
// before a publish, so the planner sees a real diff if the operation is interrupted.
//
// field must be a pointer to a field within config. Type safety is enforced by the
// compiler: field and value must have the same type F.
func SaveStateWith[F any](ctx context.Context, s *StateSaver, id string, config any, field *F, value F) {
	saved := *field
	*field = value
	s.SaveState(ctx, id, config)
	*field = saved
}

// SaveState saves the resource state. id must be the resource's identifier; on
// the first call it is recorded, and subsequent calls panic if a different id is
// passed. x must be a pointer to the same struct type as the resource's state.
// If the state is identical to what was last saved, the write is skipped.
// Failures to persist state are logged but do not abort the deployment — the
// resource already exists and aborting would not undo its creation.
func (e *StateSaver) SaveState(ctx context.Context, id string, x any) {
	if e.id == "" {
		e.id = id
	} else if e.id != id {
		panic(fmt.Sprintf("SaveState: id mismatch: expected %q, got %q", e.id, id))
	}
	xt := reflect.TypeOf(x)
	if xt != e.stateType {
		panic(fmt.Sprintf("SaveState: type mismatch: expected %v, got %v", e.stateType, xt))
	}
	b, _ := json.Marshal(x)
	if bytes.Equal(e.lastSaved, b) {
		log.Debugf(ctx, "SaveState: %s id=%s: skipping, state unchanged", e.resourceKey, id)
		return
	}
	preview := string(b)
	if len(preview) > 100 {
		preview = preview[:100]
	}
	log.Debugf(ctx, "SaveState: %s id=%s %d bytes: %s", e.resourceKey, id, len(b), preview)
	if err := e.saveFunc(e.id, b); err != nil {
		logdiag.LogError(ctx, err)
		return
	}
	e.lastSaved = b
}
