package completion

import (
	"os"
	"strings"
)

// StatusResult describes the current completion installation state.
type StatusResult struct {
	Installed bool   // true if completions are available by any method
	Method    string // "marker" | "homebrew" | "file" | ""
	FilePath  string // the file that is/would be modified
}

// Status checks whether shell completion is currently available.
func Status(shell Shell, homeDir string) (*StatusResult, error) {
	filePath := TargetFilePath(shell, homeDir)
	result := &StatusResult{FilePath: filePath}

	// Check for our marker block in the target file.
	if content, err := os.ReadFile(filePath); err == nil {
		if strings.Contains(string(content), BeginMarker) {
			result.Installed = true
			result.Method = "marker"
			return result, nil
		}
	}

	// For fish: check if the file exists at all (could be installed by a package manager).
	if shell == Fish {
		if _, err := os.Stat(filePath); err == nil {
			result.Installed = true
			result.Method = "file"
			return result, nil
		}
	}

	// For zsh: check Homebrew completions.
	if shell == Zsh {
		if p := homebrewCompletionPath(); p != "" {
			if _, err := os.Stat(p); err == nil {
				result.Installed = true
				result.Method = "homebrew"
				return result, nil
			}
		}
	}

	return result, nil
}
