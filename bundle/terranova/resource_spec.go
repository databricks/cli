package terranova

import (
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go"
)

type ReadinessEvalFunc func(*ResourceSpec, dyn.Value) (bool, error)

type ResourceSpec struct {
	DefaultPath   string
	ConfigIDField string
	ExtractIDFunc func(config dyn.Value) (string, error)

	Create CallSpec
	Update CallSpec
	Delete CallSpec
	Read   CallSpec

	Processors []Processor

	// TODO: move to []WaitConfig
	// TODO: this is not actually used yet
	ReadinessField        string
	ReadinessFieldSuccess []string
	ReadinessFieldFailure []string
	ReadinessEval         ReadinessEvalFunc
}

func (s *ResourceSpec) DoCreate(ctx context.Context, resourceID string, config dyn.Value, client *databricks.WorkspaceClient) (string, error) {
	call, err := s.Create.PrepareCall(config, resourceID)
	if err != nil {
		return resourceID, err
	}

	// TODO: cache this
	apiClient, err := client.Config.NewApiClient()
	if err != nil {
		return resourceID, err
	}

	err = call.Perform(ctx, SDKHTTPClient{apiClient})
	// if got 5xx, retry even though it may create orphaned resource. warn user about that.
	// if got 4xx, it maybe possible that orphaned resource conflicts with this one in some way. might need clean up.
	if err != nil {
		return "", fmt.Errorf("Failed to create: %s %s %d %w", call.Spec.Method, call.Path, call.StatusCode, err)
	}

	if resourceID == "" && call.Spec.ResponseIDField != "" {
		if call.ResponseID == "" {
			return "", fmt.Errorf("Failed to extract ResponseIDField=%s from create response: %s %s %d %#v", call.Spec.ResponseIDField, call.Spec.Method, call.Path, call.StatusCode, call.ResponseBody)
		}
		resourceID = call.ResponseID
	}

	return resourceID, nil
}

func (s *ResourceSpec) DoUpdate(ctx context.Context, resourceID string, configOld, config dyn.Value, client *databricks.WorkspaceClient) error {
	// TODO: cache this
	apiClient, err := client.Config.NewApiClient()
	if err != nil {
		return err
	}

	call, err := s.Update.PrepareCall(config, resourceID)
	if err != nil {
		return err
	}

	err = call.Perform(ctx, SDKHTTPClient{apiClient})
	if err != nil {
		return fmt.Errorf("Failed to update: %s %s %d %w", call.Spec.Method, call.Path, call.StatusCode, err)
	}

	return nil
}

func (r *ResourceSpec) Initialize() {
	if r.Create.Path == "" {
		r.Create.Path = r.DefaultPath
	}
	if r.Update.Path == "" {
		r.Update.Path = r.DefaultPath
	}
	if r.Delete.Path == "" {
		r.Delete.Path = r.DefaultPath
	}
	if r.Read.Path == "" {
		r.Read.Path = r.DefaultPath
	}
	if r.ReadinessEval == nil {
		r.ReadinessEval = DefaultReadinessEval
	}
}

func DefaultReadinessEval(spec *ResourceSpec, value dyn.Value) (bool, error) {
	state, ok := dyn.GetByString(value, spec.ReadinessField).AsString()
	if !ok {
		return true, fmt.Errorf("Cannot parse field %#v: missing or wrong type", spec.ReadinessField)
	}
	if slices.Contains(spec.ReadinessFieldSuccess, state) {
		return true, nil
	}
	if slices.Contains(spec.ReadinessFieldFailure, state) {
		return true, fmt.Errorf("Field %#v, value %#v", spec.ReadinessField, state)
	}
	return false, nil
}

func (spec *ResourceSpec) ExtractIDFromConfig(value dyn.Value) (string, error) {
	if spec.ExtractIDFunc != nil {
		return spec.ExtractIDFunc(value)
	}

	if spec.ConfigIDField == "" {
		return "", nil
	}

	resultValue := dyn.GetByString(value, spec.ConfigIDField)
	if !resultValue.IsValid() {
		return "", fmt.Errorf("Cannot use field %#v as resource ID: not found", spec.ConfigIDField)
	}

	result, ok := resultValue.AsString()
	// TODO: convert integer to strings
	// TODO: Validate absence of ${var.x} or ${resource.x}

	if !ok {
		return "", fmt.Errorf("Cannot use field %#v as resource ID: unexpected type %s", spec.ConfigIDField, resultValue.Kind().String())
	}

	if result == "" {
		return "", fmt.Errorf("Cannot use field %#v as resource ID: empty strings not allowed", spec.ConfigIDField)
	}

	if result == "{}" {
		return "", fmt.Errorf("Cannot use field %#v as resource ID: string '{}' not allowed", spec.ConfigIDField)
	}

	return result, nil
}

func (spec *ResourceSpec) PreprocessConfig(value dyn.Value) (dyn.Value, error) {
	return ApplyProcessors(spec.Processors, value)
}
