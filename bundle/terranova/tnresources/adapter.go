package tnresources

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/calladapt"
	"github.com/databricks/cli/libs/structs/structdiff"
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

	// DoRefresh reads and returns remote state from the backend. The return type defines schema for remote field resolution.
	// Example: func (r *ResourceJob) DoRefresh(ctx context.Context, id string) (*jobs.Job, error) {
	DoRefresh(ctx context.Context, id string) (remoteState any, e error)

	// DoDelete deletes the resource.
	// Example: func (r *ResourceJob) DoDelete(ctx context.Context, id string) error {
	DoDelete(ctx context.Context, id string) error

	// [Optional] FieldTriggers returns actions to trigger when given fields are changed.
	// Keys are field paths (e.g., ".name", ".catalog_name"). Values are actions.
	// Unspecified changed fields default to ActionTypeUpdate.
	FieldTriggers() map[string]deployplan.ActionType
}

// IResourceNoRefresh describes additional methods for resource to implement.
// Each method exists in two forms: NoRefresh (this interface) and WithRefresh (IResourceWithInterface).
// Resource can pick which signature to implement for each method individually.
type IResourceNoRefresh interface {
	// Any field named newState below has the same type as the return value of PrepareState.
	// Any field named remoteState below has the same type as the return value of DoRefresh.
	// We pass newState as a pointer but it is never nil. Changes to it will be persisted in the state, so should be used carefully.

	// DoCreate creates a new resource from the newState.
	// Example: func (r *ResourceJob) DoCreate(ctx context.Context, newState *jobs.JobSettings) (string, error) {
	DoCreate(ctx context.Context, newState any) (id string, e error)

	// DoUpdate updates the resource. ID must not change as a result of this operation.
	// Example: func (r *ResourceJob) DoUpdate(ctx context.Context, id string, newState *jobs.JobSettings) error {
	DoUpdate(ctx context.Context, id string, newState any) error

	// [Optional] DoUpdateWithID performs an update that may result in resource having a new ID
	// Example: func (r *ResourceVolume) DoUpdateWithID(ctx, id string, newState *catalog.CreateVolumeRequestContent) (string, error)
	DoUpdateWithID(ctx context.Context, id string, newState any) (string, error)

	// [Optional] WaitAfterCreate waits for the resource to become ready after creation.
	// TODO: wait status should be persisted in the state.
	WaitAfterCreate(ctx context.Context, newState any) error

	// [Optional] WaitAfterUpdate waits for the resource to become ready after update.
	WaitAfterUpdate(ctx context.Context, newState any) error
}

// IResourceWithRefresh is an alternative to IResourceNoRefresh but every method can return remoteState.
// Only use these if you get remote state for free as part of the main operation. Otherwise, prefer simpler NoRefresh variants. The state will be fetched via separate DoRefresh in this case.
// Note, resource implementations don't pick between IResourceNoRefresh and IResourceWithRefresh, they can make independent decision for each of the methods.
type IResourceWithRefresh interface {
	// DoCreate creates a new resource from the newState. Returns id of the resource and remote state.
	// Example: func (r *ResourceVolume) DoCreate(ctx context.Context, newState *catalog.CreateWarehouseRequestContent) (string, *catalog.VolumeInfo, error) {
	DoCreate(ctx context.Context, newState any) (id string, remoteState any, e error)

	// DoUpdate updates the resource. ID must not change as a result of this operation. Returns remote state.
	// Example: func (r *ResourceSchema) DoUpdate(ctx context.Context, id string, newState *catalog.CreateSchema) (*catalog.SchemaInfo, error) {
	DoUpdate(ctx context.Context, id string, newState any) (remoteState any, e error)

	// Optional: updates that may change ID. Returns new id and remote state when available.
	DoUpdateWithID(ctx context.Context, id string, newState any) (newID string, remoteState any, e error)

	// WaitAfterCreate waits for the resource to become ready after creation.
	WaitAfterCreate(ctx context.Context, newState any) (newRemoteState any, e error)

	// WaitAfterUpdate waits for the resource to become ready after update.
	WaitAfterUpdate(ctx context.Context, newState any) (newRemoteState any, e error)
}

// Adapter wraps resource implementation, validates signatures and type consistency across methods
// and provides a unified interface.
type Adapter struct {
	// Required:
	prepareState *calladapt.BoundCaller
	doRefresh    *calladapt.BoundCaller
	doDelete     *calladapt.BoundCaller
	doCreate     *calladapt.BoundCaller
	doUpdate     *calladapt.BoundCaller

	// Optional:
	doUpdateWithID  *calladapt.BoundCaller
	waitAfterCreate *calladapt.BoundCaller
	waitAfterUpdate *calladapt.BoundCaller

	fieldTriggers map[string]deployplan.ActionType
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
		prepareState:    nil,
		doRefresh:       nil,
		doDelete:        nil,
		doCreate:        nil,
		doUpdate:        nil,
		doUpdateWithID:  nil,
		waitAfterCreate: nil,
		waitAfterUpdate: nil,
		fieldTriggers:   map[string]deployplan.ActionType{},
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
		outs, err := triggerCall.Call()
		if err != nil || len(outs) != 1 {
			return nil, fmt.Errorf("failed to call FieldTriggers: %w", err)
		}
		fields := outs[0].(map[string]deployplan.ActionType)
		adapter.fieldTriggers = make(map[string]deployplan.ActionType, len(fields))
		for k, v := range fields {
			adapter.fieldTriggers[k] = v
		}
	}

	err = adapter.validate()
	if err != nil {
		return nil, err
	}

	return adapter, nil
}

func (a *Adapter) initMethods(resource any) error {
	err := calladapt.EnsureNoExtraMethods(resource, calladapt.TypeOf[IResource](), calladapt.TypeOf[IResourceNoRefresh](), calladapt.TypeOf[IResourceWithRefresh]())
	if err != nil {
		return err
	}

	a.prepareState, err = prepareCallRequired(resource, "PrepareState")
	if err != nil {
		return err
	}

	a.doRefresh, err = prepareCallRequired(resource, "DoRefresh")
	if err != nil {
		return err
	}

	a.doDelete, err = prepareCallRequired(resource, "DoDelete")
	if err != nil {
		return err
	}

	a.doCreate, err = prepareCallFromTwoVariantsRequired(resource, "DoCreate")
	if err != nil {
		return err
	}

	a.doUpdate, err = prepareCallFromTwoVariantsRequired(resource, "DoUpdate")
	if err != nil {
		return err
	}

	// Optional methods:

	a.doUpdateWithID, err = prepareCallFromTwoVariants(resource, "DoUpdateWithID")
	if err != nil {
		return err
	}

	a.waitAfterCreate, err = prepareCallFromTwoVariants(resource, "WaitAfterCreate")
	if err != nil {
		return err
	}

	a.waitAfterUpdate, err = prepareCallFromTwoVariants(resource, "WaitAfterUpdate")
	if err != nil {
		return err
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
		"DoUpdate newState", a.doUpdate.InTypes[2], stateType,
	}

	// Check if this is WithRefresh version (returns 3 values: id, remoteState, error)
	if len(a.doCreate.OutTypes) == 3 {
		validations = append(validations, "DoCreate remoteState return", a.doCreate.OutTypes[1], remoteType)
	}

	if len(a.doUpdate.OutTypes) == 2 {
		validations = append(validations, "DoUpdate remoteState return", a.doUpdate.OutTypes[0], remoteType)
	}

	if a.doUpdateWithID != nil {
		validations = append(validations, "DoUpdateWithID newState", a.doUpdateWithID.InTypes[2], stateType)
		if len(a.doUpdateWithID.OutTypes) == 3 {
			validations = append(validations, "DoUpdateWithID remoteState return", a.doUpdateWithID.OutTypes[1], remoteType)
		}
	}

	if a.waitAfterCreate != nil {
		validations = append(validations, "WaitAfterCreate newState", a.waitAfterCreate.InTypes[1], stateType)
		if len(a.waitAfterCreate.OutTypes) == 2 {
			validations = append(validations, "WaitAfterCreate remoteState return", a.waitAfterCreate.OutTypes[0], remoteType)
		}
	}

	if a.waitAfterUpdate != nil {
		validations = append(validations, "WaitAfterUpdate newState", a.waitAfterUpdate.InTypes[1], stateType)
		if len(a.waitAfterUpdate.OutTypes) == 2 {
			validations = append(validations, "WaitAfterUpdate remoteState return", a.waitAfterUpdate.OutTypes[0], remoteType)
		}
	}

	err = validateTypes(validations...)
	if err != nil {
		return err
	}

	// FieldTriggers validation
	hasUpdateWithIDTrigger := false
	for _, action := range a.fieldTriggers {
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

func (a *Adapter) DoRefresh(ctx context.Context, id string) (any, error) {
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

func (a *Adapter) DoCreate(ctx context.Context, newState any) (string, any, error) {
	if a.doCreate == nil {
		return "", nil, errors.New("internal error: DoCreate not found")
	}

	outs, err := a.doCreate.Call(ctx, newState)
	if err != nil {
		return "", nil, err
	}

	// no error checking, type is enforced via calladapt + interface
	id := outs[0].(string)

	// No refresh variant returns   (string,      err)
	// With refresh variant returns (string, any, err)
	// We normalize it to           (string, any, err)
	if len(outs) == 2 {
		// WithRefresh version
		return id, outs[1], nil
	} else {
		return id, nil, nil
	}
}

// DoUpdate updates the resource. If the implementation returns remote state,
// it will be returned as the first value; otherwise it will be nil.
func (a *Adapter) DoUpdate(ctx context.Context, id string, newState any) (any, error) {
	if a.doUpdate == nil {
		return nil, errors.New("internal error: DoUpdate not found")
	}

	outs, err := a.doUpdate.Call(ctx, id, newState)
	if err != nil {
		return nil, err
	}

	if len(outs) == 1 {
		// WithRefresh version
		return outs[0], nil
	} else {
		// NoRefresh version
		return nil, nil
	}
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

	if len(outs) == 2 {
		// WithRefresh version
		return id, outs[1], nil
	} else {
		return id, nil, nil
	}
}

// ClassifyByTriggers classifies a set of changes using FieldTriggers.
// Unspecified changed fields default to ActionTypeUpdate. Final action is the
// maximum by precedence. No changes yield ActionTypeNoop.
func (a *Adapter) ClassifyByTriggers(changes []structdiff.Change) deployplan.ActionType {
	if len(changes) == 0 {
		return deployplan.ActionTypeNoop
	}

	// Default when there are changes but no explicit trigger is update.
	result := deployplan.ActionTypeUpdate
	for _, change := range changes {
		action, ok := a.fieldTriggers[change.Path.String()]
		if !ok {
			action = deployplan.ActionTypeUpdate
		}
		if action > result {
			result = action
		}
	}
	return result
}

// WaitAfterCreate waits for the resource to become ready after creation.
// If the resource doesn't implement this method, this is a no-op.
// Returns the updated remoteState if the WithRefresh variant is implemented, otherwise returns nil
func (a *Adapter) WaitAfterCreate(ctx context.Context, newState any) (any, error) {
	if a.waitAfterCreate == nil {
		return nil, nil // no-op if not implemented
	}

	outs, err := a.waitAfterCreate.Call(ctx, newState)
	if err != nil {
		return nil, err
	}

	if len(outs) == 0 {
		// NoRefresh version
		return nil, nil
	} else {
		// WithRefresh version
		return outs[0], nil
	}
}

// WaitAfterUpdate waits for the resource to become ready after update.
// If the resource doesn't implement this method, this is a no-op.
// Returns the updated remoteState if the WithRefresh variant is implemented, otherwise returns the input remoteState.
func (a *Adapter) WaitAfterUpdate(ctx context.Context, newState any) (any, error) {
	if a.waitAfterUpdate == nil {
		return nil, nil // no-op if not implemented
	}

	outs, err := a.waitAfterUpdate.Call(ctx, newState)
	if err != nil {
		return nil, err
	}

	if len(outs) == 0 {
		// NoRefresh version
		return nil, nil
	} else {
		// WithRefresh version
		return outs[0], nil
	}
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

// prepareCallFromTwoVariants tries to prepare a call from two interface variants (NoRefresh and WithRefresh).
// Returns the caller from whichever variant works, or nil if neither works.
func prepareCallFromTwoVariants(resource any, methodName string) (*calladapt.BoundCaller, error) {
	noRefreshCaller, errNoRefresh := calladapt.PrepareCall(resource, calladapt.TypeOf[IResourceNoRefresh](), methodName)
	withRefreshCaller, errWithRefresh := calladapt.PrepareCall(resource, calladapt.TypeOf[IResourceWithRefresh](), methodName)

	// If both variants have errors, report them - these are real errors
	if errNoRefresh != nil && errWithRefresh != nil {
		return nil, fmt.Errorf("%s errors - NoRefresh: %w, WithRefresh: %w", methodName, errNoRefresh, errWithRefresh)
	}

	// Return the successful variant
	if noRefreshCaller != nil {
		return noRefreshCaller, nil
	} else if withRefreshCaller != nil {
		return withRefreshCaller, nil
	}

	return nil, nil // Neither variant found, but that might be okay for optional methods
}

// prepareCallFromTwoVariantsRequired tries to prepare a call from two interface variants and ensures one is found.
func prepareCallFromTwoVariantsRequired(resource any, methodName string) (*calladapt.BoundCaller, error) {
	caller, err := prepareCallFromTwoVariants(resource, methodName)
	if err != nil {
		return nil, err
	}
	if caller == nil {
		return nil, fmt.Errorf("%s method not found in either variant", methodName)
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
