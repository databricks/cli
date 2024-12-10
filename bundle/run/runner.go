package run

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/resources"
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

// IsRunnable returns a filter that only allows runnable resources.
func IsRunnable(ref refs.Reference) bool {
	switch ref.Resource.(type) {
	case *resources.Job, *resources.Pipeline, *resources.App:
		return true
	default:
		return false
	}
}

// ToRunner converts a resource reference to a runnable resource.
func ToRunner(b *bundle.Bundle, ref refs.Reference) (Runner, error) {
	switch resource := ref.Resource.(type) {
	case *resources.Job:
		return &jobRunner{key: key(ref.KeyWithType), bundle: b, job: resource}, nil
	case *resources.Pipeline:
		return &pipelineRunner{key: key(ref.KeyWithType), bundle: b, pipeline: resource}, nil
	case *resources.App:
		return &appRunner{
			key:    key(ref.KeyWithType),
			bundle: b,
			app:    resource,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported resource type: %T", resource)
	}
}
