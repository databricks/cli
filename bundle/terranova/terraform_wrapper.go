package terranova

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle/deploy/terraform/tfdyn"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/terraform-provider-databricks/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type ExtractIDFuncType func(t *TerraformWrapper, config dyn.Value) (string, error)

type TerraformWrapper struct {
	ConverterName  string
	CommonResource common.Resource
	ExtractIDFunc  ExtractIDFuncType
}

func (*TerraformWrapper) Initialize() {}

func (*TerraformWrapper) PreprocessConfig(config dyn.Value) (dyn.Value, error) {
	return config, nil
}

func (t *TerraformWrapper) ExtractIDFromConfig(config dyn.Value) (string, error) {
	if t.ExtractIDFunc == nil {
		return "", nil
	}
	return t.ExtractIDFunc(t, config)
}

func MakeResourceDataRaw(sch map[string]*schema.Schema, raw map[string]any) (*schema.ResourceData, error) {
	// Convert the raw map to ensure all values are compatible with Terraform's type system
	convertedRaw := make(map[string]any)
	for k, v := range raw {
		if s, ok := sch[k]; ok && (s.Type == schema.TypeList || s.Type == schema.TypeSet) {
			// If the schema expects a list/set but we have a map, wrap it in an array
			if mapVal, isMap := v.(map[string]any); isMap {
				convertedRaw[k] = []any{convertValue(mapVal)}
			} else {
				convertedRaw[k] = convertValue(v)
			}
		} else {
			convertedRaw[k] = convertValue(v)
		}
	}

	c := terraform.NewResourceConfigRaw(convertedRaw)

	sm := schema.InternalMap(sch)
	diff, err := sm.Diff(context.Background(), nil, c, nil, nil, true)
	if err != nil {
		return nil, err
	}

	result, err := sm.Data(nil, diff)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func convertValue(v any) any {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case int:
		return float64(val)
	case int32:
		return float64(val)
	case int64:
		return float64(val)
	case map[string]any:
		result := make(map[string]any)
		for k, v := range val {
			result[k] = convertValue(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = convertValue(v)
		}
		return result
	default:
		return v
	}
}

func MakeResourceDataFromDynValue(sch map[string]*schema.Schema, config dyn.Value) (*schema.ResourceData, error) {
	configMap, ok := config.AsMap()
	if !ok {
		return nil, errors.New("failed to convert config to map")
	}

	data := make(map[string]any)
	for _, pair := range configMap.Pairs() {
		key := pair.Key.MustString()
		val := pair.Value.AsAny()

		// We don't need to do schema-specific conversions here anymore
		// as convertValue in MakeResourceDataRaw will handle the conversions
		data[key] = val
	}

	return MakeResourceDataRaw(sch, data)
}

func (t *TerraformWrapper) convert(ctx context.Context, config dyn.Value) (dyn.Value, error) {
	if t.ConverterName == "" {
		return config, nil
	}

	conv, ok := tfdyn.GetConverter(t.ConverterName)
	if !ok {
		return dyn.InvalidValue, fmt.Errorf("Internal error: no such converter: %s", t.ConverterName)
	}

	return conv.ConvertDyn(ctx, config)
}

func (t *TerraformWrapper) DoCreate(ctx context.Context, resourceID string, config dyn.Value, wclient *databricks.WorkspaceClient) (string, error) {
	config, err := t.convert(ctx, config)
	if err != nil {
		return "", err
	}

	d, err := MakeResourceDataFromDynValue(t.CommonResource.Schema, config)
	if err != nil {
		return "", err
	}

	d.SetId(resourceID)

	client := &common.DatabricksClient{}
	client.SetWorkspaceClient(wclient)

	err = t.CommonResource.Create(ctx, d, client)

	// Get ID from the resource data after creation
	newID := d.Id()
	if newID != "" {
		resourceID = newID
	} else {
		// Fallback to checking "id" attribute if Id() doesn't return anything
		if setID := d.Get("id"); setID != nil {
			switch v := setID.(type) {
			case string:
				if v != "" {
					resourceID = v
				}
			case int, int64, float64:
				resourceID = fmt.Sprintf("%v", v)
			}
		}
	}

	return resourceID, err
}

func (t *TerraformWrapper) DoUpdate(ctx context.Context, resourceID string, configOld, config dyn.Value, wclient *databricks.WorkspaceClient) error {
	config, err := t.convert(ctx, config)
	if err != nil {
		return err
	}

	d, err := MakeResourceDataFromDynValue(t.CommonResource.Schema, config)
	if err != nil {
		return fmt.Errorf("failed to create resource data: %w", err)
	}

	d.SetId(resourceID)

	client := &common.DatabricksClient{}
	client.SetWorkspaceClient(wclient)

	return t.CommonResource.Update(ctx, d, client)
}
