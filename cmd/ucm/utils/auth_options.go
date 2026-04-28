package utils

import (
	"context"

	"github.com/spf13/cobra"
)

type skipPrompt int

var skipPromptKey skipPrompt

// SkipPrompt allows to skip prompt for profile configuration in MustWorkspaceClient.
//
// When calling MustWorkspaceClient we want to be able to customise if to show prompt or not.
// Since we can't change function interface, in the code we only have an access to `cmd` object.
// Command struct does not have any state flag which indicates that it's being called in completion mode and
// thus the Context object seems to be the only viable option for us to configure prompt behaviour based on
// the context it's executed from.
func SkipPrompt(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipPromptKey, true)
}

// shouldSkipPrompt returns whether or not [SkipPrompt] has been set on the specified context.
func shouldSkipPrompt(ctx context.Context) bool {
	skipPrompt, ok := ctx.Value(skipPromptKey).(bool)
	return ok && skipPrompt
}

type skipLoadUcm int

var skipLoadUcmKey skipLoadUcm

// SkipLoadUcm instructs [MustWorkspaceClient] to never try and load a ucm for configuration options.
//
// This is used for commands where we need to ensure that a ucm configuration never taints
// the authentication setup as prepared in the environment.
func SkipLoadUcm(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipLoadUcmKey, true)
}

// shouldSkipLoadUcm returns whether or not [SkipLoadUcm] has been set on the specified context.
func shouldSkipLoadUcm(ctx context.Context) bool {
	skipLoadUcm, ok := ctx.Value(skipLoadUcmKey).(bool)
	return ok && skipLoadUcm
}

func profileFlagValue(cmd *cobra.Command) (string, bool) {
	profileFlag := cmd.Flag("profile")
	if profileFlag == nil {
		return "", false
	}
	value := profileFlag.Value.String()
	return value, value != ""
}
