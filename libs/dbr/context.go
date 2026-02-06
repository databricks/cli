package dbr

import (
	"context"
	"strconv"
	"strings"
)

// key is a package-local type to use for context keys.
//
// Using an unexported type for context keys prevents key collisions across
// packages since external packages cannot create values of this type.
type key int

const (
	// dbrKey is the context key for the detection result.
	// The value of 1 is arbitrary and can be any number.
	// Other keys in the same package must have different values.
	dbrKey = key(1)
)

// ClusterType represents the type of Databricks cluster.
type ClusterType int

const (
	ClusterTypeUnknown ClusterType = iota
	ClusterTypeInteractive
	ClusterTypeServerless
)

func (t ClusterType) String() string {
	switch t {
	case ClusterTypeInteractive:
		return "interactive"
	case ClusterTypeServerless:
		return "serverless"
	default:
		return "unknown"
	}
}

// Version represents a parsed DBR version.
type Version struct {
	Type  ClusterType
	Major int
	Minor int
	Raw   string
}

// ParseVersion parses a DBR version string and returns structured version info.
// Examples:
//   - "16.3" -> Interactive, Major=16, Minor=3
//   - "client.4.9" -> Serverless, Major=4, Minor=9
func ParseVersion(version string) Version {
	result := Version{Raw: version}

	if version == "" {
		return result
	}

	// Serverless versions have "client." prefix
	if strings.HasPrefix(version, "client.") {
		result.Type = ClusterTypeServerless
		// Parse "client.X.Y" format
		parts := strings.Split(strings.TrimPrefix(version, "client."), ".")
		if len(parts) >= 1 {
			result.Major, _ = strconv.Atoi(parts[0])
		}
		if len(parts) >= 2 {
			result.Minor, _ = strconv.Atoi(parts[1])
		}
		return result
	}

	// Interactive versions are "X.Y" format
	result.Type = ClusterTypeInteractive
	parts := strings.Split(version, ".")
	if len(parts) >= 1 {
		result.Major, _ = strconv.Atoi(parts[0])
	}
	if len(parts) >= 2 {
		result.Minor, _ = strconv.Atoi(parts[1])
	}
	return result
}

type Environment struct {
	IsDbr   bool
	Version string
}

// DetectRuntime detects whether or not the current
// process is running inside a Databricks Runtime environment.
// It return a new context with the detection result set.
func DetectRuntime(ctx context.Context) context.Context {
	if v := ctx.Value(dbrKey); v != nil {
		panic("dbr.DetectRuntime called twice on the same context")
	}
	return context.WithValue(ctx, dbrKey, detect(ctx))
}

// MockRuntime is a helper function to mock the detection result.
// It returns a new context with the detection result set.
func MockRuntime(ctx context.Context, runtime Environment) context.Context {
	if v := ctx.Value(dbrKey); v != nil {
		panic("dbr.MockRuntime called twice on the same context")
	}
	return context.WithValue(ctx, dbrKey, runtime)
}

// RunsOnRuntime returns the detection result from the context.
// It expects a context returned by [DetectRuntime] or [MockRuntime].
//
// We store this value in a context to avoid having to use either
// a global variable, passing a boolean around everywhere, or
// performing the same detection multiple times.
func RunsOnRuntime(ctx context.Context) bool {
	v := ctx.Value(dbrKey)
	if v == nil {
		panic("dbr.RunsOnRuntime called without calling dbr.DetectRuntime first")
	}
	return v.(Environment).IsDbr
}

func RuntimeVersion(ctx context.Context) string {
	v := ctx.Value(dbrKey)
	if v == nil {
		panic("dbr.RuntimeVersion called without calling dbr.DetectRuntime first")
	}

	return v.(Environment).Version
}

// GetVersion returns the parsed runtime version from the context.
// It expects a context returned by [DetectRuntime] or [MockRuntime].
func GetVersion(ctx context.Context) Version {
	return ParseVersion(RuntimeVersion(ctx))
}
