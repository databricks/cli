package dbr

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContext_DetectRuntimePanics(t *testing.T) {
	ctx := context.Background()

	// Run detection.
	ctx = DetectRuntime(ctx)

	// Expect a panic if the detection is run twice.
	assert.Panics(t, func() {
		ctx = DetectRuntime(ctx)
	})
}

func TestContext_MockRuntimePanics(t *testing.T) {
	ctx := context.Background()

	// Run detection.
	ctx = MockRuntime(ctx, Environment{IsDbr: true, Version: "15.4"})

	// Expect a panic if the mock function is run twice.
	assert.Panics(t, func() {
		MockRuntime(ctx, Environment{IsDbr: true, Version: "15.4"})
	})
}

func TestContext_RunsOnRuntimePanics(t *testing.T) {
	ctx := context.Background()

	// Expect a panic if the detection is not run.
	assert.Panics(t, func() {
		RunsOnRuntime(ctx)
	})
}

func TestContext_RuntimeVersionPanics(t *testing.T) {
	ctx := context.Background()

	// Expect a panic if the detection is not run.
	assert.Panics(t, func() {
		RuntimeVersion(ctx)
	})
}

func TestContext_RunsOnRuntime(t *testing.T) {
	ctx := context.Background()

	// Run detection.
	ctx = DetectRuntime(ctx)

	// Expect no panic because detection has run.
	assert.NotPanics(t, func() {
		RunsOnRuntime(ctx)
	})
}

func TestContext_RuntimeVersion(t *testing.T) {
	ctx := context.Background()

	// Run detection.
	ctx = DetectRuntime(ctx)

	// Expect no panic because detection has run.
	assert.NotPanics(t, func() {
		RuntimeVersion(ctx)
	})
}

func TestContext_RunsOnRuntimeWithMock(t *testing.T) {
	ctx := context.Background()
	assert.True(t, RunsOnRuntime(MockRuntime(ctx, Environment{IsDbr: true, Version: "15.4"})))
	assert.False(t, RunsOnRuntime(MockRuntime(ctx, Environment{})))
}

func TestContext_RuntimeVersionWithMock(t *testing.T) {
	ctx := context.Background()
	assert.Equal(t, "15.4", RuntimeVersion(MockRuntime(ctx, Environment{IsDbr: true, Version: "15.4"})).String())
	assert.Empty(t, RuntimeVersion(MockRuntime(ctx, Environment{})).String())
}

func TestParseVersion_Serverless(t *testing.T) {
	tests := []struct {
		version       string
		expectedType  ClusterType
		expectedMajor int
		expectedMinor int
	}{
		{"client.4.9", ClusterTypeServerless, 4, 9},
		{"client.4.10", ClusterTypeServerless, 4, 10},
		{"client.3.6", ClusterTypeServerless, 3, 6},
		{"client.2", ClusterTypeServerless, 2, 0},
		{"client.2.1", ClusterTypeServerless, 2, 1},
		{"client.1", ClusterTypeServerless, 1, 0},
		{"client.1.13", ClusterTypeServerless, 1, 13},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			v := ParseVersion(tt.version)
			assert.Equal(t, tt.expectedType, v.Type)
			assert.Equal(t, tt.expectedMajor, v.Major)
			assert.Equal(t, tt.expectedMinor, v.Minor)
			assert.Equal(t, tt.version, v.Raw)
		})
	}
}

func TestParseVersion_Interactive(t *testing.T) {
	tests := []struct {
		version       string
		expectedType  ClusterType
		expectedMajor int
		expectedMinor int
	}{
		{"16.3", ClusterTypeInteractive, 16, 3},
		{"16.4", ClusterTypeInteractive, 16, 4},
		{"17.0", ClusterTypeInteractive, 17, 0},
		{"17.1", ClusterTypeInteractive, 17, 1},
		{"17.2", ClusterTypeInteractive, 17, 2},
		{"17.3", ClusterTypeInteractive, 17, 3},
		{"15.4", ClusterTypeInteractive, 15, 4},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			v := ParseVersion(tt.version)
			assert.Equal(t, tt.expectedType, v.Type)
			assert.Equal(t, tt.expectedMajor, v.Major)
			assert.Equal(t, tt.expectedMinor, v.Minor)
			assert.Equal(t, tt.version, v.Raw)
		})
	}
}

func TestParseVersion_Empty(t *testing.T) {
	v := ParseVersion("")
	assert.Equal(t, ClusterTypeUnknown, v.Type)
	assert.Equal(t, 0, v.Major)
	assert.Equal(t, 0, v.Minor)
	assert.Equal(t, "", v.Raw)
}

func TestClusterType_String(t *testing.T) {
	assert.Equal(t, "interactive", ClusterTypeInteractive.String())
	assert.Equal(t, "serverless", ClusterTypeServerless.String())
	assert.Equal(t, "unknown", ClusterTypeUnknown.String())
}

func TestVersion_String(t *testing.T) {
	v := ParseVersion("16.3")
	assert.Equal(t, "16.3", v.String())

	v = ParseVersion("client.4.9")
	assert.Equal(t, "client.4.9", v.String())

	v = ParseVersion("")
	assert.Equal(t, "", v.String())
}

func TestContext_RuntimeVersionParsed(t *testing.T) {
	ctx := context.Background()

	// Test serverless version
	serverlessCtx := MockRuntime(ctx, Environment{IsDbr: true, Version: "client.4.9"})
	v := RuntimeVersion(serverlessCtx)
	assert.Equal(t, ClusterTypeServerless, v.Type)
	assert.Equal(t, 4, v.Major)
	assert.Equal(t, 9, v.Minor)

	// Test interactive version
	interactiveCtx := MockRuntime(ctx, Environment{IsDbr: true, Version: "17.3"})
	v = RuntimeVersion(interactiveCtx)
	assert.Equal(t, ClusterTypeInteractive, v.Type)
	assert.Equal(t, 17, v.Major)
	assert.Equal(t, 3, v.Minor)
}
