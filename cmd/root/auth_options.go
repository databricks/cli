package root

import (
	"context"
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

type skipLoadBundle int

var skipLoadBundleKey skipLoadBundle

// SkipLoadBundle instructs [MustWorkspaceClient] to never try and load a bundle for configuration options.
//
// This is used for the sync command, where we need to ensure that a bundle configuration never taints
// the authentication setup as prepared in the environment (by our VS Code extension).
// Once the VS Code extension fully builds on bundles, we can remove this check again.
func SkipLoadBundle(ctx context.Context) context.Context {
	return context.WithValue(ctx, skipLoadBundleKey, true)
}

// shouldSkipLoadBundle returns whether or not [SkipLoadBundle] has been set on the specified context.
func shouldSkipLoadBundle(ctx context.Context) bool {
	skipLoadBundle, ok := ctx.Value(skipLoadBundleKey).(bool)
	return ok && skipLoadBundle
}
