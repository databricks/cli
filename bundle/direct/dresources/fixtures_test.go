package dresources

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"testing"

	"github.com/databricks/databricks-sdk-go"
)

// The shared tests of this package read the per-type data below. Each resource type registers
// its own from an init() in its test file, so a new type adds its data in its own file.

type prepareWorkspace func(ctx context.Context, client *databricks.WorkspaceClient) (any, error)

var (
	// testConfig holds the input config the shared tests prepare the state of a resource type from.
	testConfig = map[string]any{}

	// testDeps holds, for a resource type that requires other resources to exist, a function that
	// creates them and returns the input config. The CRUD tests use it instead of testConfig.
	testDeps = map[string]prepareWorkspace{}

	// withoutTestFixture lists the resource types that the shared tests exercise with a zero
	// value of the input config, with the reason.
	withoutTestFixture = map[string]string{}
)

// registerOnce adds value to registry under resourceType. It panics on a second registration,
// which would otherwise replace the first depending on the order the init()s run in.
func registerOnce[V any](registry map[string]V, registryName, resourceType string, value V) {
	if _, ok := registry[resourceType]; ok {
		panic(fmt.Sprintf("%s: resource type %q is registered more than once", registryName, resourceType))
	}
	registry[resourceType] = value
}

// registerTestConfig registers the input config of resourceType. Call it from an init().
func registerTestConfig(resourceType string, config any) {
	registerOnce(testConfig, "testConfig", resourceType, config)
}

// registerTestDeps registers the function that creates the resources resourceType requires and
// returns its input config. Call it from an init().
func registerTestDeps(resourceType string, prepare prepareWorkspace) {
	registerOnce(testDeps, "testDeps", resourceType, prepare)
}

// registerWithoutTestFixture declares that resourceType registers neither a config nor deps, and
// why the zero value of its input config is enough. Call it from an init().
func registerWithoutTestFixture(resourceType, reason string) {
	registerOnce(withoutTestFixture, "withoutTestFixture", resourceType, reason)
}

// Every adapter has a test fixture or a reason why it has none, and every registration is for a
// resource type that has an adapter.
func TestEveryAdapterHasTestFixture(t *testing.T) {
	supported := SupportedResources()

	for _, resourceType := range slices.Sorted(maps.Keys(supported)) {
		_, hasConfig := testConfig[resourceType]
		_, hasDeps := testDeps[resourceType]
		reason, optedOut := withoutTestFixture[resourceType]
		hasFixture := hasConfig || hasDeps
		if !hasFixture && !optedOut {
			t.Errorf("resource type %q has no test fixture; call registerTestConfig or registerTestDeps from the init() of its test file, or registerWithoutTestFixture with the reason", resourceType)
		}
		if hasFixture && optedOut {
			t.Errorf("resource type %q has a test fixture but is registered with registerWithoutTestFixture", resourceType)
		}
		if optedOut && reason == "" {
			t.Errorf("resource type %q is registered with registerWithoutTestFixture without a reason", resourceType)
		}
	}

	registered := map[string][]string{
		"testConfig":               slices.Collect(maps.Keys(testConfig)),
		"testDeps":                 slices.Collect(maps.Keys(testDeps)),
		"withoutTestFixture":       slices.Collect(maps.Keys(withoutTestFixture)),
		"knownMissingInRemoteType": slices.Collect(maps.Keys(knownMissingInRemoteType)),
		"knownMissingInStateType":  slices.Collect(maps.Keys(knownMissingInStateType)),
		"createRecoveryOptOut":     slices.Collect(maps.Keys(createRecoveryOptOut)),
	}
	for _, registryName := range slices.Sorted(maps.Keys(registered)) {
		for _, resourceType := range slices.Sorted(slices.Values(registered[registryName])) {
			if _, ok := supported[resourceType]; !ok {
				t.Errorf("%s has an entry for %q, which has no adapter", registryName, resourceType)
			}
		}
	}

	for _, resourceType := range slices.Sorted(maps.Keys(createRecoveryOptOut)) {
		if createRecoveryOptOut[resourceType] == "" {
			t.Errorf("resource type %q is registered with registerCreateRecoveryOptOut without a reason", resourceType)
		}
	}
}
