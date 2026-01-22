package dsc

import (
	"encoding/json"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockHandler struct {
	GetFunc    func(ctx ResourceContext, input json.RawMessage) (any, error)
	SetFunc    func(ctx ResourceContext, input json.RawMessage) error
	DeleteFunc func(ctx ResourceContext, input json.RawMessage) error
	ExportFunc func(ctx ResourceContext) (any, error)
}

func (m *MockHandler) Get(ctx ResourceContext, input json.RawMessage) (any, error) {
	if m.GetFunc != nil {
		return m.GetFunc(ctx, input)
	}
	return map[string]any{"name": "test"}, nil
}

func (m *MockHandler) Set(ctx ResourceContext, input json.RawMessage) error {
	if m.SetFunc != nil {
		return m.SetFunc(ctx, input)
	}
	return nil
}

func (m *MockHandler) Delete(ctx ResourceContext, input json.RawMessage) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(ctx, input)
	}
	return nil
}

func (m *MockHandler) Export(ctx ResourceContext) (any, error) {
	if m.ExportFunc != nil {
		return m.ExportFunc(ctx)
	}
	return []any{}, nil
}

func TestRegisterResource(t *testing.T) {
	originalRegistry := make(map[string]ResourceHandler)
	for k, v := range resourceRegistry {
		originalRegistry[k] = v
	}
	defer func() {
		resourceRegistry = originalRegistry
	}()

	resourceRegistry = make(map[string]ResourceHandler)

	handler := &MockHandler{}
	RegisterResource("TestResource", handler)

	retrieved, err := getResourceHandler("TestResource")
	require.NoError(t, err)
	assert.Equal(t, handler, retrieved)
}

func TestRegisterResourceWithMetadata(t *testing.T) {
	originalRegistry := make(map[string]ResourceHandler)
	for k, v := range resourceRegistry {
		originalRegistry[k] = v
	}
	originalMetadata := make(map[string]ResourceMetadata)
	for k, v := range metadataRegistry {
		originalMetadata[k] = v
	}
	defer func() {
		resourceRegistry = originalRegistry
		metadataRegistry = originalMetadata
	}()

	resourceRegistry = make(map[string]ResourceHandler)
	metadataRegistry = make(map[string]ResourceMetadata)

	handler := &MockHandler{}
	metadata := ResourceMetadata{
		Version:     "1.0.0",
		Description: "Test resource",
		Tags:        []string{"test"},
	}
	RegisterResourceWithMetadata("TestResource", handler, metadata)

	retrieved, err := getResourceHandler("TestResource")
	require.NoError(t, err)
	assert.Equal(t, handler, retrieved)

	retrievedMetadata, err := getResourceMetadata("TestResource")
	require.NoError(t, err)
	assert.Equal(t, metadata.Version, retrievedMetadata.Version)
	assert.Equal(t, metadata.Description, retrievedMetadata.Description)
}

func TestGetResourceHandlerNotFound(t *testing.T) {
	_, err := getResourceHandler("NonExistentResource")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown resource type")
}

func TestGetResourceMetadataNotFound(t *testing.T) {
	_, err := getResourceMetadata("NonExistentResource")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no metadata for resource type")
}

func TestListResourceTypes(t *testing.T) {
	originalRegistry := make(map[string]ResourceHandler)
	for k, v := range resourceRegistry {
		originalRegistry[k] = v
	}
	defer func() {
		resourceRegistry = originalRegistry
	}()

	resourceRegistry = make(map[string]ResourceHandler)

	handler := &MockHandler{}
	RegisterResource("ResourceA", handler)
	RegisterResource("ResourceB", handler)

	types := listResourceTypes()
	assert.Len(t, types, 2)
	assert.Contains(t, types, "ResourceA")
	assert.Contains(t, types, "ResourceB")
}

func TestBuildManifest(t *testing.T) {
	metadata := ResourceMetadata{
		Version:     "1.0.0",
		Description: "Test resource description",
		Tags:        []string{"test", "example"},
		ExitCodes:   DefaultExitCodes(),
	}

	manifest := buildManifest("Databricks/TestResource", metadata)

	assert.Equal(t, "https://aka.ms/dsc/schemas/v3/bundled/resource/manifest.json", manifest.Schema)
	assert.Equal(t, "Databricks/TestResource", manifest.Type)
	assert.Equal(t, "1.0.0", manifest.Version)
	assert.Equal(t, "Test resource description", manifest.Description)
	assert.Equal(t, []string{"test", "example"}, manifest.Tags)

	require.NotNil(t, manifest.Get)
	assert.Equal(t, "databricks", manifest.Get.Executable)
	assert.Contains(t, manifest.Get.Args, "dsc")
	assert.Contains(t, manifest.Get.Args, "get")
	assert.Contains(t, manifest.Get.Args, "--resource")
	assert.Contains(t, manifest.Get.Args, "Databricks/TestResource")

	require.NotNil(t, manifest.Set)
	assert.Contains(t, manifest.Set.Args, "set")

	require.NotNil(t, manifest.Delete)
	assert.Contains(t, manifest.Delete.Args, "delete")

	require.NotNil(t, manifest.Export)
	assert.Contains(t, manifest.Export.Args, "export")
}

func TestParseInputWithFlag(t *testing.T) {
	input := `{"name": "test"}`
	raw, err := parseInput(input)
	require.NoError(t, err)

	var result map[string]string
	err = json.Unmarshal(raw, &result)
	require.NoError(t, err)
	assert.Equal(t, "test", result["name"])
}

func TestParseInputInvalidJSON(t *testing.T) {
	_, err := parseInput("invalid json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid JSON input")
}

func TestParseInputEmpty(t *testing.T) {
	_, err := parseInput("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no JSON input provided")
}

func TestNewCommand(t *testing.T) {
	cmd := New()
	assert.Equal(t, "dsc", cmd.Use)
	assert.NotEmpty(t, cmd.Short)

	subcommands := cmd.Commands()
	names := make([]string, 0, len(subcommands))
	for _, sub := range subcommands {
		names = append(names, sub.Name())
	}

	assert.Contains(t, names, "get")
	assert.Contains(t, names, "set")
	assert.Contains(t, names, "delete")
	assert.Contains(t, names, "export")
	assert.Contains(t, names, "schema")
	assert.Contains(t, names, "manifest")
}

func TestResourceHandlerInterfaceCompliance(t *testing.T) {
	// Verify that actual handlers implement ResourceHandler
	var _ ResourceHandler = &SecretHandler{}
	var _ ResourceHandler = &SecretScopeHandler{}
	var _ ResourceHandler = &SecretAclHandler{}
	var _ ResourceHandler = &UserHandler{}
}
