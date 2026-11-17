package sandbox

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/databricks/databricks-sdk-go/apierr"
)

// resolveLocalID maps a user-typed argument to a sandbox ID using only
// the local state cache — no API calls. Resolution order:
//
//  1. Exact ID match → return arg unchanged. Anyone scripting against
//     IDs (or pasting an ID they already know) gets the fast path.
//  2. Exact name match in the cache:
//     - One match → return that sandbox's ID.
//     - Several matches → error with the candidate IDs, since the
//     server allows multiple sandboxes to share a `--name` and the
//     CLI must not silently guess which one was meant.
//  3. No match → pass the arg through. It's either an ID the cache
//     hasn't seen yet (fresh sandbox created elsewhere; the user just
//     hasn't run `sandbox list` since) or a typo. Either way the next
//     API call surfaces a clean 404 instead of us refusing locally.
//
// This is intentionally an offline shortcut: a stale cache only fails
// to find a name; it never causes the CLI to act on the wrong sandbox.
// Run `sandbox list` to refresh.
func resolveLocalID(ctx context.Context, profile, arg string) (string, error) {
	if arg == "" {
		return "", errors.New("empty sandbox identifier")
	}

	sbs := getSandboxes(ctx, profile)

	for _, s := range sbs {
		if s.ID == arg {
			return arg, nil
		}
	}

	var matches []cachedSandbox
	for _, s := range sbs {
		if s.Name != "" && s.Name == arg {
			matches = append(matches, s)
		}
	}

	switch len(matches) {
	case 0:
		return arg, nil
	case 1:
		return matches[0].ID, nil
	default:
		ids := make([]string, len(matches))
		for i, m := range matches {
			ids[i] = m.ID
		}
		return "", fmt.Errorf(
			"name %q is ambiguous in local cache; sandboxes with this name: %s — pass the ID directly",
			arg, strings.Join(ids, ", "),
		)
	}
}

// defaultSandboxHint ends the errors for a missing or stale default sandbox.
const defaultSandboxHint = "run `databricks sandbox create` to provision one, or `databricks sandbox default <id>` to set an existing sandbox as the default"

// resolveSandbox returns the ID of the sandbox named by arg, or of the
// profile's default sandbox when arg is empty, and the sandbox as the server
// reports it. The entry is nil when the lookup failed with an error other than
// not-found or a rejected request: those only warn, so a transient API hiccup
// doesn't block a connection the gateway can still route.
func resolveSandbox(ctx context.Context, api *sandboxAPI, profile, arg string) (string, *sandboxEntry, error) {
	var sandboxID string
	if arg == "" {
		if sandboxID = getDefault(ctx, profile); sandboxID == "" {
			return "", nil, errors.New("no default sandbox configured — " + defaultSandboxHint)
		}
	} else {
		resolved, err := resolveLocalID(ctx, profile, arg)
		if err != nil {
			return "", nil, err
		}
		sandboxID = resolved
	}
	if err := validateSandboxID(sandboxID); err != nil {
		return "", nil, err
	}

	sb, err := api.get(ctx, sandboxID)
	switch {
	case err == nil:
		return sandboxID, sb, nil
	case errors.Is(err, apierr.ErrNotFound) && arg == "":
		// A stale default is almost always user-correctable (deleted from
		// another machine, reaped by an admin); fail fast with an actionable
		// message.
		_ = clearDefault(ctx, profile)
		return "", nil, fmt.Errorf("saved default %q no longer exists (cleared) — %s", sandboxID, defaultSandboxHint)
	case errors.Is(err, apierr.ErrNotFound):
		return "", nil, fmt.Errorf("no sandbox named %q — `databricks sandbox list` shows available IDs", sandboxID)
	case errors.Is(err, apierr.ErrBadRequest):
		return "", nil, fmt.Errorf("failed to look up sandbox %s: %w", sandboxID, err)
	default:
		warn(ctx, fmt.Sprintf("Could not resolve sandbox %s: %v", sandboxID, err))
		return sandboxID, nil, nil
	}
}

// validateSandboxID rejects IDs that are not a single path segment. Commands
// put the ID in workspace URL paths, and the workspace decodes an escaped "/",
// so a separator or dot segment would move a request onto another API with the
// caller's credentials.
func validateSandboxID(id string) error {
	if id == "." || id == ".." || strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("invalid sandbox ID %q", id)
	}
	return nil
}
