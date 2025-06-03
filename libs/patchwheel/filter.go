package patchwheel

import (
	"context"

	"github.com/databricks/cli/libs/log"
)

// FilterLatestWheels iterates over provided wheel file paths, groups them by distribution name
// and, for every group, keeps only the wheel that has the latest version according to a best-effort
// comparison of the version strings. Returned slice preserves the order of the input slice – the
// first occurrence of the chosen wheel for every distribution is retained.
//
// The comparison is *heuristic*: the algorithm tokenises version strings into alternating numeric
// and non-numeric chunks. Numeric chunks are compared as integers, while non-numeric chunks are
// compared lexicographically. This covers common cases such as "1.2.10" > "1.2.3" and timestamps
// added via calculateNewVersion (e.g. "1.2.3+1741091696…" > "1.2.3"). It does not attempt to
// implement the full PEP 440 specification, which is unnecessary for the dynamic versions
// produced by this package.
func FilterLatestWheels(ctx context.Context, paths []string) []string {
	// Build output incrementally, preserving the order of the *chosen* wheels.
	out := make([]string, 0, len(paths))

	// distribution -> index in out slice
	bestIdx := make(map[string]int)

	for _, p := range paths {
		info, err := ParseWheelFilename(p)
		if err != nil {
			// Unparsable: always keep.
			out = append(out, p)
			continue
		}

		if idx, seen := bestIdx[info.Distribution]; !seen {
			// First wheel for this distribution.
			bestIdx[info.Distribution] = len(out)
			out = append(out, p)
			continue
		} else {
			// Compare against the current winner.
			winnerPath := out[idx]
			winnerInfo, _ := ParseWheelFilename(winnerPath) // guaranteed parseable

			if compareVersion(info.Version, winnerInfo.Version) > 0 {
				// Current wheel wins: replace earlier entry in-place.
				log.Debugf(ctx, "Skipping wheel %s (older than %s)", winnerPath, p)
				out[idx] = p
			} else {
				// Current wheel loses.
				log.Debugf(ctx, "Skipping wheel %s (older than %s)", p, winnerPath)
			}
		}
	}

	return out
}
