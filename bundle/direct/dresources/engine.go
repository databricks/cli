package dresources

import (
	"errors"
	"fmt"
	"reflect"
)

// Engine provides state persistence to resource implementations.
// Pass it to DoCreate or DoUpdate to save intermediate state before long-running
// wait operations, so the resource is not orphaned if deployment is interrupted.
type Engine struct {
	id        string
	stateType reflect.Type
	saveFunc  func(id string, x any) error
}

// NewEngine creates an Engine with the given state type and save function.
// The framework calls this before invoking DoCreate or DoUpdate.
func NewEngine(stateType reflect.Type, saveFunc func(id string, x any) error) *Engine {
	return &Engine{id: "", stateType: stateType, saveFunc: saveFunc}
}

// NewNopEngine creates an Engine that discards all saves. Use in tests.
func NewNopEngine(stateType reflect.Type) *Engine {
	return NewEngine(stateType, func(_ string, _ any) error { return nil })
}

// SetID sets the resource id for subsequent SaveState calls.
// Must be called before SaveState during DoCreate; for DoUpdate the Engine is
// pre-configured with the existing id.
func (e *Engine) SetID(id string) {
	e.id = id
}

// SaveState saves the resource state. x must be of the same pointer-to-struct
// type as the resource's state type. Returns an error if SetID was not called.
func (e *Engine) SaveState(x any) error {
	if e.id == "" {
		return errors.New("SaveState: id not set, call SetID first")
	}
	xt := reflect.TypeOf(x)
	if xt != e.stateType {
		return fmt.Errorf("SaveState: type mismatch: expected %v, got %v", e.stateType, xt)
	}
	return e.saveFunc(e.id, x)
}
