package run

import (
	"context"
	"fmt"
	"reflect"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	refs "github.com/databricks/cli/bundle/resources"
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

	// Restart the underlying workflow by cancelling any existing runs before
	// starting a new one.
	Restart(ctx context.Context, opts *Options) (output.RunOutput, error)

	// Cancel the underlying workflow.
	Cancel(ctx context.Context) error

	// Runners support parsing and completion of additional positional arguments.
	argsHandler
}

// OptionsResolver is implemented by runners whose options must be resolved
// against the bundle after positional arguments are parsed and before the run starts.
type OptionsResolver interface {
	ResolveOptions(ctx context.Context, opts *Options) error
}

type runnerFactory func(b *bundle.Bundle, k key, resource config.ConfigResource) Runner

// runnerFactories is keyed by the resource's Go type, so lookups do not
// depend on the order in which runner files register.
var runnerFactories = map[reflect.Type]runnerFactory{}

// registerRunner makes resources of type T runnable. Call it from the
// runner's file in func init().
func registerRunner[T config.ConfigResource](newRunner func(b *bundle.Bundle, k key, resource T) Runner) {
	t := reflect.TypeFor[T]()
	if _, ok := runnerFactories[t]; ok {
		panic(fmt.Sprintf("runner for %v registered twice", t))
	}
	runnerFactories[t] = func(b *bundle.Bundle, k key, resource config.ConfigResource) Runner {
		return newRunner(b, k, resource.(T))
	}
}

// IsRunnable returns a filter that only allows runnable resources.
func IsRunnable(ref refs.Reference) bool {
	_, ok := runnerFactories[reflect.TypeOf(ref.Resource)]
	return ok
}

// ToRunner converts a resource reference to a runnable resource.
func ToRunner(b *bundle.Bundle, ref refs.Reference) (Runner, error) {
	newRunner, ok := runnerFactories[reflect.TypeOf(ref.Resource)]
	if !ok {
		return nil, fmt.Errorf("unsupported resource type: %T", ref.Resource)
	}
	return newRunner(b, key(ref.KeyWithType), ref.Resource), nil
}
