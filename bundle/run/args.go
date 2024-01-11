package run

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// argsHandler defines the (unexported) interface for the runners in this
// package to implement to handle context-specific positional arguments.
//
// For jobs, this means:
//   - If a job uses job parameters: parse positional arguments into key-value pairs
//     and pass them as job parameters.
//   - If a job does not use job parameters AND only has Spark Python tasks:
//     pass through the positional arguments as a list of Python parameters.
//   - If a job does not use job parameters AND only has notebook tasks:
//     parse arguments into key-value pairs and pass them as notebook parameters.
//   - ...
//
// In all cases, we may be able to provide context-aware argument completions.
type argsHandler interface {
	// Parse additional positional arguments.
	ParseArgs(args []string, opts *Options) error

	// Complete additional positional arguments.
	CompleteArgs(args []string, toComplete string) ([]string, cobra.ShellCompDirective)
}

// nopArgsHandler is a no-op implementation of [argsHandler].
// It returns an error if any positional arguments are present and doesn't complete anything.
type nopArgsHandler struct{}

func (nopArgsHandler) ParseArgs(args []string, opts *Options) error {
	if len(args) == 0 {
		return nil
	}

	return fmt.Errorf("received %d unexpected positional arguments", len(args))
}

func (nopArgsHandler) CompleteArgs(args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	return nil, cobra.ShellCompDirectiveNoFileComp
}

// argsToKeyValueMap parses key-value pairs from the specified arguments.
//
// It accepts these formats:
//   - `--key=value`
//   - `--key`, `value`
//
// Remaining arguments are returned as-is.
func argsToKeyValueMap(args []string) (map[string]string, []string) {
	kv := make(map[string]string)
	key := ""
	tail := args

	for i, arg := range args {
		// If key is set; use the next argument as value.
		if key != "" {
			kv[key] = arg
			key = ""
			tail = args[i+1:]
			continue
		}

		if strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(arg[2:], "=", 2)
			if len(parts) == 2 {
				kv[parts[0]] = parts[1]
				tail = args[i+1:]
				continue
			}

			// Use this argument as key, the next as value.
			key = parts[0]
			continue
		}

		// If we cannot interpret it; return here.
		break
	}

	return kv, tail
}

// genericParseKeyValueArgs parses key-value pairs from the specified arguments.
// If there are any positional arguments left, it returns an error.
func genericParseKeyValueArgs(args []string) (map[string]string, error) {
	kv, args := argsToKeyValueMap(args)
	if len(args) > 0 {
		return nil, fmt.Errorf("received %d unexpected positional arguments", len(args))
	}

	return kv, nil
}

// genericCompleteKeyValueArgs completes key-value pairs from the specified arguments.
// Completion options that are already specified are skipped.
func genericCompleteKeyValueArgs(args []string, toComplete string, options []string) ([]string, cobra.ShellCompDirective) {
	// If the string to complete contains an equals sign, then we are
	// completing the value part (which we don't know here).
	if strings.Contains(toComplete, "=") {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Remove already completed key/value pairs.
	kv, args := argsToKeyValueMap(args)

	// If the list of remaining args is empty, return possible completions.
	if len(args) == 0 {
		var completions []string
		for _, option := range options {
			// Skip options that have already been specified.
			if _, ok := kv[option]; ok {
				continue
			}
			completions = append(completions, fmt.Sprintf("--%s=", option))
		}
		// Note: we include cobra.ShellCompDirectiveNoSpace to suggest including
		// the value part right after the equals sign.
		return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}
