package version

import (
	"fmt"
	"strings"
)

var (
	// Version is the current version of the application
	// Set during build with -ldflags "-X github.com/appdotbuild/go-mcp/pkg/version.Version=x.y.z"
	Version = "dev"

	// Commit is the git commit hash
	// Set during build with -ldflags "-X github.com/appdotbuild/go-mcp/pkg/version.Commit=..."
	Commit = "unknown"

	// BuildTime is the time when the binary was built
	// Set during build with -ldflags "-X github.com/appdotbuild/go-mcp/pkg/version.BuildTime=..."
	BuildTime = "unknown"
)

// GetVersion returns the full version string
func GetVersion() string {
	if Version == "dev" {
		return fmt.Sprintf("dev (commit: %s, built: %s)", Commit, BuildTime)
	}
	// Check if version already has 'v' prefix
	version := Version
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return fmt.Sprintf("%s (commit: %s, built: %s)", version, Commit, BuildTime)
}
