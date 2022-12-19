package run

import (
	"context"
	"fmt"
	"strings"

	"github.com/databricks/bricks/bundle"
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

	// Run the underlying worklow.
	Run(ctx context.Context, opts *Options) error
}

// Collect collects a list of runners given a list of arguments.
//
// Its behavior is as follows:
//  1. If no arguments are specified, it returns a runner for the only resource in the bundle.
//  2. If multiple arguments are specified, for each argument:
//     2.1. Try to find a resource with <key> identical to the argument.
//     2.2. Try to find a resource with <type>.<key> identical to the argument.
//
// If an argument resolves to multiple resources, it returns an error.
func Collect(b *bundle.Bundle, args []string) ([]Runner, error) {
	keyOnly, keyWithType := ResourceKeys(b)
	if len(keyWithType) == 0 {
		return nil, fmt.Errorf("bundle defines no resources")
	}

	var out []Runner

	// If the bundle contains only a single resource, we know what to run.
	if len(args) == 0 {
		if len(keyWithType) != 1 {
			return nil, fmt.Errorf("bundle defines multiple resources; please specify resource to run")
		}
		for _, runners := range keyWithType {
			if len(runners) != 1 {
				// This invariant is covered by [ResourceKeys].
				panic("length of []run.Runner must be 1")
			}
			out = append(out, runners[0])
		}
		return out, nil
	}

	for _, arg := range args {
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
		out = append(out, runners[0])
	}

	return out, nil
}
