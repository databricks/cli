package workspace

import (
	"github.com/appdotbuild/go-mcp/pkg/pathutil"
)

// validatePath ensures the given user path is safe and within the base directory
func validatePath(baseDir, userPath string) (string, error) {
	return pathutil.ValidatePath(baseDir, userPath)
}
