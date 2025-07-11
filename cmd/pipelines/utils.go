package pipelines

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/resources"
	"github.com/databricks/cli/bundle/run"
	"github.com/databricks/cli/libs/cmdio"
)

// promptRunArgument prompts the user to select a resource to run.
// Copied from cmd/bundle/run.go
func promptRunArgument(ctx context.Context, b *bundle.Bundle) (string, error) {
	// Compute map of "Human readable name of resource" -> "resource key".
	inv := make(map[string]string)
	for k, ref := range resources.Completions(b, run.IsRunnable) {
		title := fmt.Sprintf("%s: %s", ref.Description.SingularTitle, ref.Resource.GetName())
		inv[title] = k
	}

	key, err := cmdio.Select(ctx, inv, "Resource to run")
	if err != nil {
		return "", err
	}

	return key, nil
}

// resolveRunArgument resolves the resource key to run.
// It returns the remaining arguments to pass to the runner, if applicable.
// Copied from cmd/bundle/run.go
func resolveRunArgument(ctx context.Context, b *bundle.Bundle, args []string) (string, []string, error) {
	// If no arguments are specified, prompt the user to select something to run.
	if len(args) == 0 && cmdio.IsPromptSupported(ctx) {
		key, err := promptRunArgument(ctx, b)
		if err != nil {
			return "", nil, err
		}
		return key, args, nil
	}

	if len(args) < 1 {
		return "", nil, errors.New("expected a KEY of the resource to run")
	}

	return args[0], args[1:], nil
}

// keyToRunner converts a resource key to a runner.
// Copied from cmd/bundle/run.go
func keyToRunner(b *bundle.Bundle, arg string) (run.Runner, error) {
	// Locate the resource to run.
	ref, err := resources.Lookup(b, arg, run.IsRunnable)
	if err != nil {
		return nil, err
	}

	// Convert the resource to a runnable resource.
	runner, err := run.ToRunner(b, ref)
	if err != nil {
		return nil, err
	}

	return runner, nil
}
