package run

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/run/output"
)

type key string

func (k key) Key() string {
	return string(k)
}

// Runner defines the interface for a runnable resource (or workload).
type Runner interface {
	// Key returns the fully qualified (unique) identifier for this runnable resource.
	// This is used for showing the user hints w.r.t. disambiguation.
	Key() string

	// Name returns the resource's name, if defined.
	Name() string

	// Run the underlying worklow.
	Run(ctx context.Context, opts *Options) (output.RunOutput, error)

	// Cancel the underlying workflow.
	Cancel(ctx context.Context) error
}

// Find locates a runner matching the specified argument.
//
// Its behavior is as follows:
//  1. Try to find a resource with <key> identical to the argument.
//  2. Try to find a resource with <type>.<key> identical to the argument.
//
// If an argument resolves to multiple resources, it returns an error.
func Find(b *bundle.Bundle, arg string) (Runner, error) {
	keyOnly, keyWithType := ResourceKeys(b)
	if len(keyWithType) == 0 {
		return nil, fmt.Errorf("bundle defines no resources")
	}

	runners, ok := keyOnly[arg]
	if !ok {
		runners, ok = keyWithType[arg]
		if !ok {
			return nil, fmt.Errorf("no such resource: %s", arg)
		}
	}

	if len(runners) != 1 {
		var keys []string
		for _, runner := range runners {
			keys = append(keys, runner.Key())
		}
		return nil, fmt.Errorf("ambiguous: %s (can resolve to all of %s)", arg, strings.Join(keys, ", "))
	}

	return runners[0], nil
}
