package tnresources

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/calladapt"
	"github.com/databricks/cli/libs/structdiff"
	"github.com/databricks/databricks-sdk-go"
)

// IResource represents the unified interface for resource adapters.
// The resources don't actually implement this interface, but implement the same methods with signatures with concrete types.
type IResource interface {
	// New returns a new implementation instance for a given resource.
	// Note, this is a class method, it will be called with nil receiver.
	// The return value must be a pointer to a specific instance of the resource implementation, e.g. *ResourceJob.
	// Single instance is reused across all instances, so it must not store any resource-specific state.
	New(client *databricks.WorkspaceClient) any

	// PrepareConfig converts resource's config as defined by bundle schema to the concrete type used by create/update and persisted in the state.
	// Example: func (*ResourceJob) PrepareConfig(input *resources.Job) *jobs.JobSettings
	PrepareConfig(input any) any

	// DoDelete deletes the resource.
	// Example: func (r *ResourceJob) DoDelete(ctx context.Context, id string) error {
	DoDelete(ctx context.Context, id string) error

	// DoCreate creates a new resource from the config.
	// Example: func (r *ResourceJob) DoCreate(ctx context.Context, config *jobs.JobSettings) (string, error) {
	DoCreate(ctx context.Context, config any) (id string, e error)

	// DoUpdate updates the resource. ID must not change as a result of this operation.
	// Example: func (r *ResourceJob) DoUpdate(ctx context.Context, id string, config *jobs.JobSettings) error {
	DoUpdate(ctx context.Context, id string, config any) error

	// [Optional] DoUpdateWithID performs an update that may result in resource having a new ID
	// Example: func (r *ResourceVolume) DoUpdateWithID(ctx, id string, config *catalog.CreateVolumeRequestContent) (string, error)
	DoUpdateWithID(ctx context.Context, id string, config any) (string, error)

	// [Optional] WaitAfterCreate waits for the resource to become ready after creation.
	// TODO: wait status should be persisted in the state.
	WaitAfterCreate(ctx context.Context, config any) error

	// [Optional] WaitAfterUpdate waits for the resource to become ready after update.
	WaitAfterUpdate(ctx context.Context, config any) error

	// [Optional] RecreateFields returns a list of fields that will cause resource recreation if changed
	RecreateFields() []string

	// [Optional] ClassifyChanges provides non-default change classification.
	// Default is to consider any change "an update".
	// Note, RecreateFields takes priority over this. If recreate is detected via RecreateFields, this is not going to be called.
	ClassifyChanges(changes []structdiff.Change) deployplan.ActionType
}

type Adapter struct {
	// Required:
	new           *calladapt.BoundCaller
	prepareConfig *calladapt.BoundCaller
	doDelete      *calladapt.BoundCaller
	doCreate      *calladapt.BoundCaller
	doUpdate      *calladapt.BoundCaller

	// Optional:
	doUpdateWithID  *calladapt.BoundCaller
	waitAfterCreate *calladapt.BoundCaller
	waitAfterUpdate *calladapt.BoundCaller
	classifyChanges *calladapt.BoundCaller

	recreateFields map[string]struct{}
}

func NewAdapter(typedNil any, client *databricks.WorkspaceClient) (*Adapter, error) {
	newCall, err := prepareCallRequired(typedNil, calladapt.TypeOf[IResource](), "New")
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
		new:             nil,
		prepareConfig:   nil,
		doDelete:        nil,
		doCreate:        nil,
		doUpdate:        nil,
		doUpdateWithID:  nil,
		waitAfterCreate: nil,
		waitAfterUpdate: nil,
		classifyChanges: nil,
		recreateFields:  map[string]struct{}{},
	}

	err = adapter.initMethods(impl)
	if err != nil {
		return nil, err
	}

	// Load optional RecreateFields method from the unified interface
	recreateCall, err := calladapt.PrepareCall(impl, calladapt.TypeOf[IResource](), "RecreateFields")
	if err != nil {
		return nil, err
	}
	if recreateCall != nil {
		outs, err := recreateCall.Call()
		if err != nil || len(outs) != 1 {
			return nil, fmt.Errorf("failed to call RecreateFields: %w", err)
		}
		fields := outs[0].([]string)
		adapter.recreateFields = make(map[string]struct{}, len(fields))
		for _, field := range fields {
			adapter.recreateFields[field] = struct{}{}
		}
	}

	err = adapter.validate()
	if err != nil {
		return nil, err
	}

	return adapter, nil
}

func (a *Adapter) initMethods(resource any) error {
	it := calladapt.TypeOf[IResource]()

	err := calladapt.EnsureNoExtraMethods(resource, it)
	if err != nil {
		return err
	}
	a.new, err = prepareCallRequired(resource, it, "New")
	if err != nil {
		return err
	}

	a.prepareConfig, err = prepareCallRequired(resource, it, "PrepareConfig")
	if err != nil {
		return err
	}

	a.doDelete, err = prepareCallRequired(resource, it, "DoDelete")
	if err != nil {
		return err
	}

	a.doCreate, err = prepareCallRequired(resource, it, "DoCreate")
	if err != nil {
		return err
	}

	a.doUpdate, err = prepareCallRequired(resource, it, "DoUpdate")
	if err != nil {
		return err
	}

	// Optional methods:

	a.doUpdateWithID, err = calladapt.PrepareCall(resource, it, "DoUpdateWithID")
	if err != nil {
		return err
	}

	a.waitAfterCreate, err = calladapt.PrepareCall(resource, it, "WaitAfterCreate")
	if err != nil {
		return err
	}

	a.waitAfterUpdate, err = calladapt.PrepareCall(resource, it, "WaitAfterUpdate")
	if err != nil {
		return err
	}

	a.classifyChanges, err = calladapt.PrepareCall(resource, it, "ClassifyChanges")
	return err
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
	configType := a.ConfigType()
	err := validatePointerToStruct(configType, "config type")
	if err != nil {
		return err
	}

	validations := []any{
		"PrepareConfig return", a.prepareConfig.OutTypes[0], configType,
		"DoCreate config", a.doCreate.InTypes[1], configType,
		"DoUpdate config", a.doUpdate.InTypes[2], configType,
	}

	if a.doUpdateWithID != nil {
		validations = append(validations, "DoUpdateWithID config", a.doUpdateWithID.InTypes[2], configType)
	}

	if a.waitAfterCreate != nil {
		validations = append(validations, "WaitAfterCreate config", a.waitAfterCreate.InTypes[1], configType)
	}

	if a.waitAfterUpdate != nil {
		validations = append(validations, "WaitAfterUpdate config", a.waitAfterUpdate.InTypes[1], configType)
	}

	err = validateTypes(validations...)
	if err != nil {
		return err
	}

	if a.doUpdateWithID != nil && a.classifyChanges == nil {
		return errors.New("if DoUpdateWithID is present, should have implement ClassifyChanges")
	}

	return nil
}

func (a *Adapter) InputConfigType() reflect.Type {
	return a.prepareConfig.InTypes[0]
}

func (a *Adapter) ConfigType() reflect.Type {
	return a.prepareConfig.OutTypes[0]
}

func (a *Adapter) New(client *databricks.WorkspaceClient) (any, error) {
	outs, err := a.new.Call(client)
	if err != nil {
		return nil, err
	}
	if len(outs) != 1 {
		return nil, fmt.Errorf("internal error: New returned %d values, expected 1", len(outs))
	}
	return outs[0], nil
}

func (a *Adapter) PrepareConfig(input any) (any, error) {
	outs, err := a.prepareConfig.Call(input)
	if err != nil {
		return nil, err
	}
	if len(outs) != 1 {
		return nil, fmt.Errorf("internal error: PrepareConfig returned %d values, expected 1", len(outs))
	}
	return outs[0], nil
}

func (a *Adapter) DoDelete(ctx context.Context, id string) error {
	outs, err := a.doDelete.Call(ctx, id)
	if err != nil {
		return err
	}
	if len(outs) != 0 {
		return fmt.Errorf("internal error: DoDelete returned %d values, expected 0", len(outs))
	}
	return nil
}

func (a *Adapter) DoCreate(ctx context.Context, config any) (string, error) {
	outs, err := a.doCreate.Call(ctx, config)
	if err != nil {
		return "", err
	}

	// no error checking, type is enforced via calladapt + interface
	id := outs[0].(string)
	return id, nil
}

// DoUpdate updates the resource.
func (a *Adapter) DoUpdate(ctx context.Context, id string, config any) error {
	_, err := a.doUpdate.Call(ctx, id, config)
	return err
}

// HasClassifyChanges returns true if the resource implements ClassifyChanges method.
func (a *Adapter) HasClassifyChanges() bool {
	return a.classifyChanges != nil
}

// HasDoUpdateWithID returns true if the resource implements DoUpdateWithID method.
func (a *Adapter) HasDoUpdateWithID() bool {
	return a.doUpdateWithID != nil
}

// DoUpdateWithID updates the resource and may change its ID. Returns newID.
func (a *Adapter) DoUpdateWithID(ctx context.Context, oldID string, config any) (string, error) {
	if a.doUpdateWithID == nil {
		return "", errors.New("internal error: DoUpdateWithID not found")
	}

	outs, err := a.doUpdateWithID.Call(ctx, oldID, config)
	if err != nil {
		return "", err
	}

	id := outs[0].(string)
	return id, nil
}

// MustRecreate checks if any of the changes require resource recreation.
func (a *Adapter) MustRecreate(changes []structdiff.Change) bool {
	if len(a.recreateFields) == 0 {
		return false
	}
	for _, change := range changes {
		if _, ok := a.recreateFields[change.Path.String()]; ok {
			return true
		}
	}
	return false
}

// ClassifyChanges calls the resource's ClassifyChanges method if implemented.
func (a *Adapter) ClassifyChanges(changes []structdiff.Change) (deployplan.ActionType, error) {
	if a.classifyChanges == nil {
		return "", errors.New("internal error: ClassifyChanges not implemented")
	}
	outs, err := a.classifyChanges.Call(changes)
	if err != nil {
		return "", err
	}
	result := outs[0].(deployplan.ActionType)
	return result, nil
}

// WaitAfterCreate waits for the resource to become ready after creation.
// If the resource doesn't implement this method, this is a no-op.
func (a *Adapter) WaitAfterCreate(ctx context.Context, config any) error {
	if a.waitAfterCreate == nil {
		return nil // no-op if not implemented
	}

	_, err := a.waitAfterCreate.Call(ctx, config)
	return err
}

// WaitAfterUpdate waits for the resource to become ready after update.
// If the resource doesn't implement this method, this is a no-op.
func (a *Adapter) WaitAfterUpdate(ctx context.Context, config any) error {
	if a.waitAfterUpdate == nil {
		return nil // no-op if not implemented
	}

	_, err := a.waitAfterUpdate.Call(ctx, config)
	return err
}

// prepareCallRequired prepares a call and ensures the method is found.
func prepareCallRequired(resource any, interfaceType reflect.Type, methodName string) (*calladapt.BoundCaller, error) {
	caller, err := calladapt.PrepareCall(resource, interfaceType, methodName)
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
