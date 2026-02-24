package completion

import (
	"os"
	"path/filepath"
)

// Install configures shell completion for the given shell. homeDir is used
// as the base for RC file resolution (typically os.UserHomeDir()).
// Returns the file path modified and whether it was already installed.
func Install(shell Shell, homeDir string) (filePath string, alreadyInstalled bool, err error) {
	status, err := Status(shell, homeDir)
	if err != nil {
		return TargetFilePath(shell, homeDir), false, err
	}
	filePath = status.FilePath

	// For fish, any existing file counts as "already installed" — we don't
	// overwrite files that may have been installed by a package manager.
	// For RC-based shells, only our marker block counts.
	if shell == Fish && status.Installed {
		return filePath, true, nil
	}
	if status.Method == "marker" {
		return filePath, true, nil
	}

	if shell == Fish {
		return installFish(filePath, shell)
	}
	return installRC(filePath, shell)
}

// installFish handles the file-drop model for fish completions.
// The caller must check Status before calling this — existence checks are not
// repeated here.
func installFish(filePath string, shell Shell) (string, bool, error) {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return filePath, false, err
	}

	return filePath, false, os.WriteFile(filePath, []byte(ShimContent(shell)), 0o644)
}

// installRC handles the RC file model for bash, zsh, and powershell.
// The caller must check Status before calling this — marker checks are not
// repeated here.
func installRC(filePath string, shell Shell) (string, bool, error) {
	var content []byte
	var perm os.FileMode = 0o644

	if info, err := os.Stat(filePath); err == nil {
		perm = info.Mode()
		content, err = os.ReadFile(filePath)
		if err != nil {
			return filePath, false, err
		}
	}

	// Create parent directory if needed (e.g. for PowerShell profiles).
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return filePath, false, err
	}

	// Ensure a leading newline before the block if the file doesn't end with one.
	shim := ShimContent(shell)
	if len(content) > 0 && content[len(content)-1] != '\n' {
		shim = "\n" + shim
	}

	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, perm)
	if err != nil {
		return filePath, false, err
	}
	defer f.Close()

	if _, err := f.WriteString(shim); err != nil {
		return filePath, false, err
	}

	return filePath, false, nil
}
