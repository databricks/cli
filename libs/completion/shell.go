package completion

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Shell represents a supported shell type.
type Shell string

const (
	Bash        Shell = "bash"
	Zsh         Shell = "zsh"
	Fish        Shell = "fish"
	PowerShell  Shell = "powershell"
	PowerShell5 Shell = "powershell5"
)

const (
	// BeginMarker is the start of the completion block in RC files.
	BeginMarker = "# BEGIN databricks-cli completion"
	// EndMarker is the end of the completion block in RC files.
	EndMarker = "# END databricks-cli completion"
)

// DisplayName returns a human-readable name for the shell.
func (s Shell) DisplayName() string {
	switch s {
	case PowerShell:
		return "powershell (pwsh 7+)"
	case PowerShell5:
		return "powershell5 (Windows PowerShell 5.1)"
	default:
		return string(s)
	}
}

// DetectShell returns the shell to use. If flagValue is non-empty, it validates
// and returns it. Otherwise it auto-detects from the environment.
func DetectShell(flagValue string) (Shell, error) {
	if flagValue != "" {
		return validateShellFlag(flagValue)
	}

	shellEnv := os.Getenv("SHELL")
	if shellEnv != "" {
		return shellFromPath(shellEnv)
	}

	if runtime.GOOS == "windows" {
		return detectWindowsShell()
	}

	return "", errors.New("could not detect shell: $SHELL is not set. Use --shell to specify your shell")
}

// validateShellFlag validates a user-provided --shell flag value.
func validateShellFlag(value string) (Shell, error) {
	shell := Shell(strings.ToLower(value))

	switch shell {
	case Bash, Zsh, Fish, PowerShell, PowerShell5:
	default:
		return "", fmt.Errorf("unsupported shell %q: supported shells are bash, zsh, fish, powershell, powershell5", value)
	}

	if shell == PowerShell5 && runtime.GOOS != "windows" {
		return "", errors.New("--shell powershell5 is only supported on Windows")
	}

	if shell == PowerShell && runtime.GOOS == "windows" {
		if _, err := exec.LookPath("pwsh"); err != nil {
			return "", errors.New("PowerShell 7+ (pwsh) was not found on PATH. Use --shell powershell5 for Windows PowerShell 5.1, or install pwsh from https://aka.ms/powershell")
		}
	}

	return shell, nil
}

// shellFromPath extracts the shell name from a path like /bin/bash or /usr/bin/zsh.
func shellFromPath(path string) (Shell, error) {
	name := strings.ToLower(filepath.Base(path))

	switch {
	case strings.Contains(name, "bash"):
		return Bash, nil
	case strings.Contains(name, "zsh"):
		return Zsh, nil
	case strings.Contains(name, "fish"):
		return Fish, nil
	case name == "pwsh", name == "pwsh.exe":
		return PowerShell, nil
	case name == "powershell", name == "powershell.exe":
		if runtime.GOOS == "windows" {
			return PowerShell5, nil
		}
	}

	return "", fmt.Errorf("unsupported shell %q: supported shells are bash, zsh, fish, powershell, powershell5", name)
}

// detectWindowsShell attempts to find PowerShell on Windows.
func detectWindowsShell() (Shell, error) {
	if _, err := exec.LookPath("pwsh"); err == nil {
		return PowerShell, nil
	}
	if _, err := exec.LookPath("powershell.exe"); err == nil {
		return PowerShell5, nil
	}
	return "", errors.New("could not detect shell: no supported shell found on PATH. Use --shell to specify your shell")
}

// TargetFilePath returns the file that will be modified for the given shell.
func TargetFilePath(shell Shell, homeDir string) string {
	switch shell {
	case Bash:
		return bashProfilePath(homeDir)
	case Zsh:
		return filepath.Join(homeDir, ".zshrc")
	case Fish:
		return filepath.Join(homeDir, ".config", "fish", "completions", "databricks.fish")
	case PowerShell:
		return powershellProfilePath(homeDir)
	case PowerShell5:
		return filepath.Join(homeDir, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
	default:
		return ""
	}
}

// bashProfilePath returns the appropriate bash profile path for the current OS.
// On macOS, Terminal.app and iTerm2 launch login shells that read ~/.bash_profile.
// On Linux, interactive shells read ~/.bashrc.
// Falls back to the other if the primary choice doesn't exist.
func bashProfilePath(homeDir string) string {
	primary := ".bashrc"
	fallback := ".bash_profile"
	if runtime.GOOS == "darwin" {
		primary = ".bash_profile"
		fallback = ".bashrc"
	}

	primaryPath := filepath.Join(homeDir, primary)
	if _, err := os.Stat(primaryPath); err == nil {
		return primaryPath
	}

	fallbackPath := filepath.Join(homeDir, fallback)
	if _, err := os.Stat(fallbackPath); err == nil {
		return fallbackPath
	}

	// Neither exists; return the primary (it will be created).
	return primaryPath
}

// powershellProfilePath returns the pwsh 7+ profile path.
// See: https://learn.microsoft.com/en-us/powershell/module/microsoft.powershell.core/about/about_profiles
func powershellProfilePath(homeDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(homeDir, "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1")
	}
	return filepath.Join(homeDir, ".config", "powershell", "Microsoft.PowerShell_profile.ps1")
}

// ShimContent returns the completion shim block for the given shell, including markers.
func ShimContent(shell Shell) string {
	var evalLine string
	switch shell {
	case Bash:
		evalLine = `eval "$(databricks completion bash)"`
	case Zsh:
		evalLine = `eval "$(databricks completion zsh)"`
	case Fish:
		evalLine = "databricks completion fish | source"
	case PowerShell, PowerShell5:
		evalLine = "databricks completion powershell | Out-String | Invoke-Expression"
	}

	return BeginMarker + "\n" + evalLine + "\n" + EndMarker + "\n"
}

// homebrewCompletionPath returns the path to Homebrew-installed zsh completions
// for databricks, or empty string if not found.
func homebrewCompletionPath() string {
	prefix := os.Getenv("HOMEBREW_PREFIX")
	if prefix == "" {
		// Check common defaults.
		// See: https://docs.brew.sh/Installation
		for _, p := range []string{"/opt/homebrew", "/usr/local"} {
			if _, err := os.Stat(filepath.Join(p, "bin/brew")); err == nil {
				prefix = p
				break
			}
		}
	}
	if prefix == "" {
		return ""
	}
	return filepath.Join(prefix, "share/zsh/site-functions/_databricks")
}
