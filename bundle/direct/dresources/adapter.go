package dresources

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/calladapt"
	"github.com/databricks/cli/libs/structs/structdiff"
	"github.com/databricks/cli/libs/structs/structtrie"
	"github.com/databricks/databricks-sdk-go"
)

// IResource describes core methods for the resource implementation.
// The resources don't actually implement this interface, but implement the same methods with signatures with concrete types.
// The resources need to have all of the methods on IResource that are not marked [Optional].
type IResource interface {
	// New returns a new implementation instance for a given resource.
	// Note, this is a class method, it will be called with nil receiver.
	// The return value must be a pointer to a specific instance of the resource implementation, e.g. *ResourceJob.
	// Single instance is reused across all instances, so it must not store any resource-specific state.
	New(client *databricks.WorkspaceClient) any

	// PrepareState converts resource's config as defined by bundle schema to the concrete type used by create/update and persisted in the state.
	// Example: func (*ResourceJob) PrepareState(input *resources.Job) *jobs.JobSettings
	PrepareState(input any) any

	// [Required if type(remoteState) != type(state)] RemapState adapts remote state to local state type.
	// The adapted remote state will then be compared with newState to detect remote drift.
	// Adaptation is not necessary (but possible) if types already match.
	// Example: func (*ResourceJob) RemapState(jobs *jobs.Job) *jobs.JobSettings
	RemapState(input any) any

	// DoRead reads and returns remote state from the backend. The return type defines schema for remote field resolution.
	// Example: func (r *ResourceJob) DoRead(ctx context.Context, id string) (*jobs.Job, error)
	DoRead(ctx context.Context, id string) (remoteState any, e error)

	// DoDelete deletes the resource.
	// Example: func (r *ResourceJob) DoDelete(ctx context.Context, id string) error
	DoDelete(ctx context.Context, id string) error

	// [Optional] FieldTriggers returns actions to trigger when given fields are changed.
	// Keys are field paths (e.g., "name", "catalog_name"). Values are actions.
	// Unspecified changed fields default to ActionTypeUpdate.
	//
	// FieldTriggers(true) is applied on every change between state (last deployed config)
	// and new state (current config) to determine action based on config changes.
	//
	// FieldTriggers(false) is called on every change between state and remote state to
	// determine action based on remote drift.
	//
	// Note: these functions are called once per resource implementation initialization,
	// not once per resource.
	FieldTriggers(isLocal bool) map[string]deployplan.ActionType
	// [Optional] ClassifyChange classifies a change using custom logic.
	// The isLocal parameter indicates whether this is a local change (true) or remote change (false).
	ClassifyChange(change structdiff.Change, remoteState any, isLocal bool) (deployplan.ActionType, error)

	// DoCreate creates a new resource from the newState. Returns id of the resource and optionally remote state.
	// If remote state is available as part of the operation, return it; otherwise return nil.
	// Example: func (r *ResourceVolume) DoCreate(ctx context.Context, newState *catalog.CreateVolumeRequestContent) (string, *catalog.VolumeInfo, error)
	DoCreate(ctx context.Context, newState any) (id string, remoteState any, e error)

	// [Optional] DoUpdate updates the resource. ID must not change as a result of this operation. Returns optionally remote state.
	// If remote state is available as part of the operation, return it; otherwise return nil.
	// Example: func (r *ResourceSchema) DoUpdate(ctx context.Context, id string, newState *catalog.CreateSchema, changes *deployplan.Changes) (*catalog.SchemaInfo, error)
	DoUpdate(ctx context.Context, id string, newState any, changes *deployplan.Changes) (remoteState any, e error)

	// [Optional] DoUpdateWithID performs an update that may result in resource having a new ID. Returns new id and optionally remote state.
	DoUpdateWithID(ctx context.Context, id string, newState any) (newID string, remoteState any, e error)

	// [Optional] DoResize resizes the resource. Only supported by clusters
	DoResize(ctx context.Context, id string, newState any) error

	// [Optional] WaitAfterCreate waits for the resource to become ready after creation. Returns optionally updated remote state.
	// TODO: wait status should be persisted in the state.
	WaitAfterCreate(ctx context.Context, newState any) (remoteState any, e error)

	// [Optional] WaitAfterUpdate waits for the resource to become ready after update. Returns optionally updated remote state.
	WaitAfterUpdate(ctx context.Context, newState any) (remoteState any, e error)

	// [Optional] KeyedSlices returns a map from path patterns to KeyFunc for comparing slices by key instead of by index.
	// Example: func (*ResourcePermissions) KeyedSlices(state *PermissionsState) map[string]any
	KeyedSlices() map[string]any
}

// Adapter wraps resource implementation, validates signatures and type consistency across methods
// and provides a unified interface.
type Adapter struct {
	// Required:
	prepareState *calladapt.BoundCaller
	remapState   *calladapt.BoundCaller
	doRefresh    *calladapt.BoundCaller
	doDelete     *calladapt.BoundCaller
	doCreate     *calladapt.BoundCaller

	// Optional:
	doUpdate        *calladapt.BoundCaller
	doUpdateWithID  *calladapt.BoundCaller
	waitAfterCreate *calladapt.BoundCaller
	waitAfterUpdate *calladapt.BoundCaller
	classifyChange  *calladapt.BoundCaller
	doResize        *calladapt.BoundCaller

	fieldTriggersLocal  map[string]deployplan.ActionType
	fieldTriggersRemote map[string]deployplan.ActionType
	// keyedSlices         map[string]any
	keyedSliceTrie *structtrie.Node
}

func NewAdapter(typedNil any, client *databricks.WorkspaceClient) (*Adapter, error) {
	newCall, err := prepareCallRequired(typedNil, "New")
	if err != nil {
		return nil, err
	}
	outs, err := newCall.Call(client)
	if err != nil {
		return nil, err
	}
	if len(outs) != 1 {
		return nil, fmt.Errorf("internal error: New returned %d values, expected 1", len(outs))
	}
	impl := outs[0]
	adapter := &Adapter{
		prepareState:        nil,
		remapState:          nil,
		doRefresh:           nil,
		doDelete:            nil,
		doCreate:            nil,
		doUpdate:            nil,
		doUpdateWithID:      nil,
		doResize:            nil,
		waitAfterCreate:     nil,
		waitAfterUpdate:     nil,
		classifyChange:      nil,
		fieldTriggersLocal:  map[string]deployplan.ActionType{},
		fieldTriggersRemote: map[string]deployplan.ActionType{},
		keyedSlices:         nil,
	}

	err = adapter.initMethods(impl)
	if err != nil {
		return nil, err
	}

	// Load optional FieldTriggers method from the unified interface
	triggerCall, err := calladapt.PrepareCall(impl, calladapt.TypeOf[IResource](), "FieldTriggers")
	if err != nil {
		return nil, err
	}
	if triggerCall != nil {
		// Validate FieldTriggers signature: func(bool) map[string]deployplan.ActionType
		if len(triggerCall.InTypes) != 1 || triggerCall.InTypes[0] != reflect.TypeOf(false) {
			return nil, errors.New("FieldTriggers must take a single bool parameter (isLocal)")
		}
		if len(triggerCall.OutTypes) != 1 {
			return nil, errors.New("FieldTriggers must return a single value")
		}
		expectedReturnType := reflect.TypeOf(map[string]deployplan.ActionType{})
		if triggerCall.OutTypes[0] != expectedReturnType {
			return nil, fmt.Errorf("FieldTriggers must return map[string]deployplan.ActionType, got %v", triggerCall.OutTypes[0])
		}

		// Call with isLocal=true for local triggers
		adapter.fieldTriggersLocal, err = loadFieldTriggers(triggerCall, true)
		if err != nil {
			return nil, err
		}

		// Call with isLocal=false for remote triggers
		adapter.fieldTriggersRemote, err = loadFieldTriggers(triggerCall, false)
		if err != nil {
			return nil, err
		}
	}

	err = adapter.validate()
	if err != nil {
		return nil, err
	}

	return adapter, nil
}

// loadFieldTriggers calls FieldTriggers with isLocal parameter and returns the resulting map.
func loadFieldTriggers(triggerCall *calladapt.BoundCaller, isLocal bool) (map[string]deployplan.ActionType, error) {
	outs, err := triggerCall.Call(isLocal)
	if err != nil || len(outs) != 1 {
		return nil, fmt.Errorf("failed to call FieldTriggers(%v): %w", isLocal, err)
	}
	fields := outs[0].(map[string]deployplan.ActionType)
	result := make(map[string]deployplan.ActionType, len(fields))
	maps.Copy(result, fields)
	return result, nil
}

// loadKeyedSlices validates and calls KeyedSlices method, returning the resulting map.
func loadKeyedSlices(call *calladapt.BoundCaller) (map[string]any, error) {
	outs, err := call.Call()
	if err != nil {
		return nil, fmt.Errorf("failed to call KeyedSlices: %w", err)
	}
	result := outs[0].(map[string]any)
	return result, nil
}

func (a *Adapter) initMethods(resource any) error {
	err := calladapt.EnsureNoExtraMethods(resource, calladapt.TypeOf[IResource]())
	if err != nil {
		return err
	}
	a.prepareState, err = prepareCallRequired(resource, "PrepareState")
	if err != nil {
		return err
	}

	// RemapState is optional when remote type already matches state type.
	a.remapState, err = calladapt.PrepareCall(resource, calladapt.TypeOf[IResource](), "RemapState")
	if err != nil {
		return err
	}

	a.doRefresh, err = prepareCallRequired(resource, "DoRead")
	if err != nil {
		return err
	}

	a.doDelete, err = prepareCallRequired(resource, "DoDelete")
	if err != nil {
		return err
	}

	a.doCreate, err = prepareCallRequired(resource, "DoCreate")
	if err != nil {
		return err
	}

	// Optional methods with varying signatures:

	a.doUpdate, err = calladapt.PrepareCall(resource, calladapt.TypeOf[IResource](), "DoUpdate")
	if err != nil {
		return err
	}

	a.doUpdateWithID, err = calladapt.PrepareCall(resource, calladapt.TypeOf[IResource](), "DoUpdateWithID")
	if err != nil {
		return err
	}

	a.waitAfterCreate, err = calladapt.PrepareCall(resource, calladapt.TypeOf[IResource](), "WaitAfterCreate")
	if err != nil {
		return err
	}

	a.waitAfterUpdate, err = calladapt.PrepareCall(resource, calladapt.TypeOf[IResource](), "WaitAfterUpdate")
	if err != nil {
		return err
	}

	a.classifyChange, err = calladapt.PrepareCall(resource, calladapt.TypeOf[IResource](), "ClassifyChange")
	if err != nil {
		return err
	}

	a.doResize, err = calladapt.PrepareCall(resource, calladapt.TypeOf[IResource](), "DoResize")
	if err != nil {
		return err
	}

	keyedSlicesCall, err := calladapt.PrepareCall(resource, calladapt.TypeOf[IResource](), "KeyedSlices")
	if err != nil {
		return err
	}
	if keyedSlicesCall != nil {
		a.keyedSlices, err = loadKeyedSlices(keyedSlicesCall)
		if err != nil {
			return err
		}
		if len(a.keyedSlices) > 0 {
			typed := make(map[string]structdiff.KeyFunc, len(a.keyedSlices))
			for pattern, fn := range a.keyedSlices {
				typed[pattern] = fn
			}
			a.keyedSliceTrie, err = structdiff.BuildSliceKeyTrie(typed)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// validateTypes validates type matches for variadic triples of (name, actual, expected).
func validateTypes(triples ...any) error {
	if len(triples)%3 != 0 {
		return errors.New("validateTypes requires arguments in triples of (name, actual, expected)")
	}

	for i := 0; i < len(triples); i += 3 {
		name := triples[i].(string)
		actual := triples[i+1].(reflect.Type)
		expected := triples[i+2].(reflect.Type)

		if actual != expected {
			return fmt.Errorf("%s type mismatch: expected %v, got %v", name, expected, actual)
		}
	}
	return nil
}

func (a *Adapter) validate() error {
	stateType := a.StateType()
	err := validatePointerToStruct(stateType, "state type")
	if err != nil {
		return err
	}

	remoteType := a.RemoteType()
	err = validatePointerToStruct(remoteType, "remote type")
	if err != nil {
		return err
	}

	validations := []any{
		"PrepareState return", a.prepareState.OutTypes[0], stateType,
		"DoCreate newState", a.doCreate.InTypes[1], stateType,
	}

	// If RemapState is implemented, validate its signature.
	// Otherwise require remote type to equal state type so remapping isn't needed.
	if a.remapState != nil {
		validations = append(validations,
			"RemapState input", a.remapState.InTypes[0], remoteType,
			"RemapState return", a.remapState.OutTypes[0], stateType,
		)
	} else if remoteType != stateType {
		return fmt.Errorf("RemapState method not found and remote type %v must match state type %v", remoteType, stateType)
	}

	// Validate DoCreate: must return (string, remoteType, error)
	if len(a.doCreate.OutTypes) != 3 {
		return fmt.Errorf("DoCreate must return (string, remoteType, error), got %d return values", len(a.doCreate.OutTypes))
	}
	validations = append(validations, "DoCreate remoteState return", a.doCreate.OutTypes[1], remoteType)

	// Validate DoUpdate: must return (remoteType, error) if implemented
	if a.doUpdate != nil {
		validations = append(validations, "DoUpdate newState", a.doUpdate.InTypes[2], stateType)
		if len(a.doUpdate.OutTypes) != 2 {
			return fmt.Errorf("DoUpdate must return (remoteType, error), got %d return values", len(a.doUpdate.OutTypes))
		}
		validations = append(validations, "DoUpdate remoteState return", a.doUpdate.OutTypes[0], remoteType)
	}

	if a.doResize != nil {
		validations = append(validations, "DoResize newState", a.doResize.InTypes[2], stateType)
	}

	if a.doUpdateWithID != nil {
		validations = append(validations, "DoUpdateWithID newState", a.doUpdateWithID.InTypes[2], stateType)
		// DoUpdateWithID must return (string, remoteType, error)
		if len(a.doUpdateWithID.OutTypes) != 3 {
			return fmt.Errorf("DoUpdateWithID must return (string, remoteType, error), got %d return values", len(a.doUpdateWithID.OutTypes))
		}
		validations = append(validations, "DoUpdateWithID remoteState return", a.doUpdateWithID.OutTypes[1], remoteType)
	}

	if a.waitAfterCreate != nil {
		validations = append(validations, "WaitAfterCreate newState", a.waitAfterCreate.InTypes[1], stateType)
		// WaitAfterCreate must return (remoteType, error)
		if len(a.waitAfterCreate.OutTypes) != 2 {
			return fmt.Errorf("WaitAfterCreate must return (remoteType, error), got %d return values", len(a.waitAfterCreate.OutTypes))
		}
		validations = append(validations, "WaitAfterCreate remoteState return", a.waitAfterCreate.OutTypes[0], remoteType)
	}

	if a.waitAfterUpdate != nil {
		validations = append(validations, "WaitAfterUpdate newState", a.waitAfterUpdate.InTypes[1], stateType)
		// WaitAfterUpdate must return (remoteType, error)
		if len(a.waitAfterUpdate.OutTypes) != 2 {
			return fmt.Errorf("WaitAfterUpdate must return (remoteType, error), got %d return values", len(a.waitAfterUpdate.OutTypes))
		}
		validations = append(validations, "WaitAfterUpdate remoteState return", a.waitAfterUpdate.OutTypes[0], remoteType)
	}

	if a.classifyChange != nil {
		validations = append(validations,
			"ClassifyChange remoteState", a.classifyChange.InTypes[1], remoteType,
			"ClassifyChange isLocal", a.classifyChange.InTypes[2], reflect.TypeOf(false),
		)
	}

	err = validateTypes(validations...)
	if err != nil {
		return err
	}

	// FieldTriggers validation
	hasUpdateWithIDTrigger := false
	for _, action := range a.fieldTriggersLocal {
		if action == deployplan.ActionTypeUpdateWithID {
			hasUpdateWithIDTrigger = true
		}
	}
	for _, action := range a.fieldTriggersRemote {
		if action == deployplan.ActionTypeUpdateWithID {
			hasUpdateWithIDTrigger = true
		}
	}
	if hasUpdateWithIDTrigger && a.doUpdateWithID == nil {
		return errors.New("FieldTriggers includes update_with_id but DoUpdateWithID is not implemented")
	}
	if a.doUpdateWithID != nil && !hasUpdateWithIDTrigger {
		return errors.New("DoUpdateWithID is implemented but FieldTriggers lacks update_with_id trigger")
	}

	return nil
}

func (a *Adapter) InputConfigType() reflect.Type {
	return a.prepareState.InTypes[0]
}

func (a *Adapter) StateType() reflect.Type {
	return a.prepareState.OutTypes[0]
}

func (a *Adapter) RemoteType() reflect.Type {
	return a.doRefresh.OutTypes[0]
}

func (a *Adapter) PrepareState(input any) (any, error) {
	outs, err := a.prepareState.Call(input)
	if err != nil {
		return nil, err
	}
	return outs[0], nil
}

func (a *Adapter) RemapState(remoteState any) (any, error) {
	if a.remapState == nil {
		return remoteState, nil
	}

	outs, err := a.remapState.Call(remoteState)
	if err != nil {
		return nil, err
	}
	return outs[0], nil
}

func (a *Adapter) DoRead(ctx context.Context, id string) (any, error) {
	outs, err := a.doRefresh.Call(ctx, id)
	if err != nil {
		return nil, err
	}
	return outs[0], nil
}

func (a *Adapter) DoDelete(ctx context.Context, id string) error {
	_, err := a.doDelete.Call(ctx, id)
	if err != nil {
		return err
	}
	return nil
}

// normalizeNilPointer converts a nil pointer wrapped in an interface to a nil interface.
// This is needed because reflection can return a typed nil pointer as a non-nil interface.
func normalizeNilPointer(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr && rv.IsNil() {
		return nil
	}
	return v
}

func (a *Adapter) DoCreate(ctx context.Context, newState any) (string, any, error) {
	outs, err := a.doCreate.Call(ctx, newState)
	if err != nil {
		return "", nil, err
	}

	id := outs[0].(string)
	remoteState := normalizeNilPointer(outs[1])
	return id, remoteState, nil
}

// HasDoUpdate returns true if the resource implements DoUpdate method.
func (a *Adapter) HasDoUpdate() bool {
	return a.doUpdate != nil
}

// DoUpdate updates the resource with information about changes computed during plan.
// Returns remote state if available, otherwise nil.
func (a *Adapter) DoUpdate(ctx context.Context, id string, newState any, changes *deployplan.Changes) (any, error) {
	if a.doUpdate == nil {
		return nil, errors.New("internal error: DoUpdate not found")
	}

	outs, err := a.doUpdate.Call(ctx, id, newState, changes)
	if err != nil {
		return nil, err
	}

	remoteState := normalizeNilPointer(outs[0])
	return remoteState, nil
}

// HasDoUpdateWithID returns true if the resource implements DoUpdateWithID method.
func (a *Adapter) HasDoUpdateWithID() bool {
	return a.doUpdateWithID != nil
}

// DoUpdateWithID updates the resource and may change its ID. Returns newID and remoteState if available.
func (a *Adapter) DoUpdateWithID(ctx context.Context, oldID string, newState any) (string, any, error) {
	if a.doUpdateWithID == nil {
		return "", nil, errors.New("internal error: DoUpdateWithID not found")
	}

	outs, err := a.doUpdateWithID.Call(ctx, oldID, newState)
	if err != nil {
		return "", nil, err
	}

	id := outs[0].(string)
	remoteState := normalizeNilPointer(outs[1])
	return id, remoteState, nil
}

func (a *Adapter) DoResize(ctx context.Context, id string, newState any) error {
	if a.doResize == nil {
		return errors.New("internal error: DoResize not found")
	}

	_, err := a.doResize.Call(ctx, id, newState)
	return err
}

// classifyByTriggers classifies a change using FieldTriggers.
// Defaults to ActionTypeUpdate.
// The isLocal parameter determines which trigger map to use:
// - isLocal=true uses triggers from FieldTriggers(true)
// - isLocal=false uses triggers from FieldTriggers(false)
func (a *Adapter) classifyByTriggers(change structdiff.Change, isLocal bool) deployplan.ActionType {
	var triggers map[string]deployplan.ActionType
	if isLocal {
		triggers = a.fieldTriggersLocal
	} else {
		triggers = a.fieldTriggersRemote
	}

	action, ok := triggers[change.Path.String()]
	if ok {
		return action
	}
	return deployplan.ActionTypeUpdate
}

// WaitAfterCreate waits for the resource to become ready after creation.
// If the resource doesn't implement this method, this is a no-op.
// Returns the updated remoteState if available, otherwise returns nil
func (a *Adapter) WaitAfterCreate(ctx context.Context, newState any) (any, error) {
	if a.waitAfterCreate == nil {
		return nil, nil // no-op if not implemented
	}

	outs, err := a.waitAfterCreate.Call(ctx, newState)
	if err != nil {
		return nil, err
	}

	remoteState := normalizeNilPointer(outs[0])
	return remoteState, nil
}

// WaitAfterUpdate waits for the resource to become ready after update.
// If the resource doesn't implement this method, this is a no-op.
// Returns the updated remoteState if available, otherwise returns nil.
func (a *Adapter) WaitAfterUpdate(ctx context.Context, newState any) (any, error) {
	if a.waitAfterUpdate == nil {
		return nil, nil // no-op if not implemented
	}

	outs, err := a.waitAfterUpdate.Call(ctx, newState)
	if err != nil {
		return nil, err
	}

	remoteState := normalizeNilPointer(outs[0])
	return remoteState, nil
}

// ClassifyChange classifies a change using custom logic or FieldTriggers.
// The isLocal parameter determines whether this is a local or remote change:
// - isLocal=true: classifying local changes (user modifications)
// - isLocal=false: classifying remote changes (drift detection)
func (a *Adapter) ClassifyChange(change structdiff.Change, remoteState any, isLocal bool) (deployplan.ActionType, error) {
	actionType := deployplan.ActionTypeUndefined

	// If ClassifyChange is implemented, use it.
	if a.classifyChange != nil {
		outs, err := a.classifyChange.Call(change, remoteState, isLocal)
		if err != nil {
			return deployplan.ActionTypeUndefined, err
		}
		actionType = outs[0].(deployplan.ActionType)
	}

	// If ClassifyChange is not implemented or is implemented but returns undefined, use FieldTriggers.
	if actionType == deployplan.ActionTypeUndefined {
		return a.classifyByTriggers(change, isLocal), nil
	}
	return actionType, nil
}

// KeyedSlices returns a map from path patterns to KeyFunc for comparing slices by key.
// If the resource doesn't implement KeyedSlices, returns nil.
func (a *Adapter) KeyedSlices() map[string]any {
	return a.keyedSlices
}

func (a *Adapter) KeyedSliceTrie() *structtrie.Node {
	return a.keyedSliceTrie
}

// prepareCallRequired prepares a call and ensures the method is found.
func prepareCallRequired(resource any, methodName string) (*calladapt.BoundCaller, error) {
	caller, err := calladapt.PrepareCall(resource, calladapt.TypeOf[IResource](), methodName)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", methodName, err)
	}
	if caller == nil {
		return nil, fmt.Errorf("%s method not found", methodName)
	}
	return caller, nil
}

// validatePointerToStruct checks if the type is a pointer to a struct.
func validatePointerToStruct(t reflect.Type, context string) error {
	if t == nil {
		return fmt.Errorf("%s not set", context)
	}
	if t.Kind() != reflect.Ptr {
		return fmt.Errorf("%s must be a pointer, got %s", context, t.Kind())
	}
	if t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("%s must be a pointer to struct, got pointer to %s", context, t.Elem().Kind())
	}
	return nil
}
