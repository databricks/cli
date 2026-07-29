package installer

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/useragent"
)

// aiDevKitStateDir is the marker directory the ai-dev-kit installer
// (github.com/databricks-solutions/ai-dev-kit) writes into both the project root
// and the global home (or $AIDEVKIT_HOME).
const aiDevKitStateDir = ".ai-dev-kit"

// aiDevKitVersionFile is the version file the ai-dev-kit installer writes under aiDevKitStateDir.
// See https://github.com/databricks-solutions/ai-dev-kit install.sh.
const aiDevKitVersionFile = "version"

// aiDevKitHomeEnv overrides the global ai-dev-kit install location.
const aiDevKitHomeEnv = "AIDEVKIT_HOME"

// AiDevKitVersion returns the installed ai-dev-kit version and whether it is
// installed, sanitized for the user agent.
func AiDevKitVersion(ctx context.Context) (string, bool) {
	installed := false
	for _, dir := range aiDevKitStateDirs(ctx) {
		version, ok := readAiDevKitVersion(dir)
		if !ok {
			continue
		}
		installed = true
		// A blank marker still means installed; keep scanning so a real version
		// in a lower-precedence scope isn't masked by an empty higher one.
		if version != "" {
			return version, true
		}
	}
	return "", installed
}

// aiDevKitStateDirs returns the candidate ai-dev-kit state directories to probe,
// project scope first. A scope whose base path can't be resolved is skipped
// rather than guessed.
func aiDevKitStateDirs(ctx context.Context) []string {
	var dirs []string
	if cwd, err := os.Getwd(); err == nil {
		dirs = append(dirs, filepath.Join(cwd, aiDevKitStateDir))
	}
	// AIDEVKIT_HOME already points at the install root, so the marker lives
	// directly under it; otherwise it defaults to ~/.ai-dev-kit.
	if home := env.Get(ctx, aiDevKitHomeEnv); home != "" {
		dirs = append(dirs, home)
	} else if home, err := env.UserHomeDir(ctx); err == nil {
		dirs = append(dirs, filepath.Join(home, aiDevKitStateDir))
	}
	return dirs
}

// readAiDevKitVersion reads the version marker under dir, reporting whether it
// exists. A present-but-blank marker reports ("", true).
func readAiDevKitVersion(dir string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(dir, aiDevKitVersionFile))
	if err != nil {
		return "", false
	}
	// Sanitize the first line: the value is set outside our control and
	// useragent panics on anything but alphanumerics/semver.
	firstLine, _, _ := strings.Cut(string(data), "\n")
	return useragent.Sanitize(strings.TrimSpace(firstLine)), true
}
