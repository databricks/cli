// Package target resolves Lakebase Postgres targets (provisioned instances and
// autoscaling endpoints) into the host, credential, and SDK metadata that
// callers need to open a connection. It is shared by `cmd/psql` and the
// `experimental postgres query` command so that both speak the same SDK.
package target

import (
	"errors"
	"fmt"
	"strings"
)

const (
	// PathSegmentProjects is the leading path segment that identifies an
	// autoscaling resource path. Provisioned instance names never start with it.
	PathSegmentProjects  = "projects"
	pathSegmentBranches  = "branches"
	pathSegmentEndpoints = "endpoints"
)

// AutoscalingSpec is a partial or full specification for an autoscaling endpoint.
// Empty fields signal "auto-select if exactly one exists, otherwise error".
type AutoscalingSpec struct {
	ProjectID  string
	BranchID   string
	EndpointID string
}

// Choice is a single candidate returned alongside an AmbiguousError so callers
// can either render the list to the user or prompt interactively.
type Choice struct {
	ID          string
	DisplayName string
}

// AmbiguousError is returned by AutoSelect* helpers when the SDK returns more
// than one candidate and the caller did not specify which one to pick.
//
// Callers that have a TTY (e.g. `databricks psql`) can use errors.As to detect
// this and prompt interactively. Callers that are non-interactive (e.g. the
// scriptable `postgres query` command) propagate it as a plain error: the
// formatted message already enumerates the choices.
type AmbiguousError struct {
	// Kind identifies what was ambiguous: "project", "branch", or "endpoint".
	Kind string
	// Parent is the SDK resource name that contained the ambiguity (e.g.
	// "projects/foo" when listing branches), or empty when listing projects.
	Parent string
	// FlagHint is the flag a user would set to disambiguate (e.g. "--branch").
	FlagHint string
	// Choices enumerates the candidates returned by the SDK.
	Choices []Choice
}

func (e *AmbiguousError) Error() string {
	plural := map[string]string{
		"project":  "projects",
		"branch":   "branches",
		"endpoint": "endpoints",
		"instance": "instances",
	}[e.Kind]
	if plural == "" {
		plural = e.Kind
	}

	var sb strings.Builder
	if e.Parent == "" {
		fmt.Fprintf(&sb, "multiple %s found; specify %s:", plural, e.FlagHint)
	} else {
		fmt.Fprintf(&sb, "multiple %s found in %s; specify %s:", plural, e.Parent, e.FlagHint)
	}
	for _, c := range e.Choices {
		sb.WriteString("\n  - ")
		sb.WriteString(c.ID)
		if c.DisplayName != "" && c.DisplayName != c.ID {
			fmt.Fprintf(&sb, " (%s)", c.DisplayName)
		}
	}
	return sb.String()
}

// ParseAutoscalingPath extracts project, branch, and endpoint IDs from a
// resource path. Accepts partial paths:
//
//	projects/foo
//	projects/foo/branches/bar
//	projects/foo/branches/bar/endpoints/baz
//
// Returns an error if the path is malformed or does not start with "projects/".
func ParseAutoscalingPath(input string) (AutoscalingSpec, error) {
	parts := strings.Split(input, "/")

	if len(parts) < 2 || parts[0] != PathSegmentProjects {
		return AutoscalingSpec{}, fmt.Errorf("invalid resource path: %s", input)
	}
	if parts[1] == "" {
		return AutoscalingSpec{}, errors.New("invalid resource path: missing project ID")
	}
	spec := AutoscalingSpec{ProjectID: parts[1]}

	if len(parts) > 2 {
		if len(parts) < 4 || parts[2] != pathSegmentBranches {
			return AutoscalingSpec{}, errors.New("invalid resource path: expected 'branches' after project")
		}
		if parts[3] == "" {
			return AutoscalingSpec{}, errors.New("invalid resource path: missing branch ID")
		}
		spec.BranchID = parts[3]
	}

	if len(parts) > 4 {
		if len(parts) < 6 || parts[4] != pathSegmentEndpoints {
			return AutoscalingSpec{}, errors.New("invalid resource path: expected 'endpoints' after branch")
		}
		if parts[5] == "" {
			return AutoscalingSpec{}, errors.New("invalid resource path: missing endpoint ID")
		}
		spec.EndpointID = parts[5]
	}

	if len(parts) > 6 {
		return AutoscalingSpec{}, fmt.Errorf("invalid resource path: trailing components after endpoint: %s", input)
	}

	return spec, nil
}

// ExtractID returns the value following component in a resource name.
// ExtractID("projects/foo/branches/bar", "branches") returns "bar".
// Returns the original name unchanged if component is not found.
func ExtractID(name, component string) string {
	parts := strings.Split(name, "/")
	for i := range len(parts) - 1 {
		if parts[i] == component {
			return parts[i+1]
		}
	}
	return name
}

// IsAutoscalingPath reports whether s is an autoscaling resource path
// (i.e. starts with "projects/"). Provisioned instance names never do.
func IsAutoscalingPath(s string) bool {
	return strings.HasPrefix(s, PathSegmentProjects+"/")
}
