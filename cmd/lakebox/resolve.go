package lakebox

import (
	"context"
	"errors"
	"fmt"
	"strings"
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
//     hasn't run `lakebox list` since) or a typo. Either way the next
//     API call surfaces a clean 404 instead of us refusing locally.
//
// This is intentionally an offline shortcut: a stale cache only fails
// to find a name; it never causes the CLI to act on the wrong sandbox.
// Run `lakebox list` to refresh.
func resolveLocalID(ctx context.Context, profile, arg string) (string, error) {
	if arg == "" {
		return "", errors.New("empty lakebox identifier")
	}

	sbs := getSandboxes(ctx, profile)

	// ID-first so that a sandbox whose --name happens to collide with
	// another sandbox's ID never gets mistakenly resolved by name.
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
