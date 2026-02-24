package completion

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectShellFromEnv(t *testing.T) {
	tests := []struct {
		name     string
		envShell string
		expected Shell
	}{
		{"bash from /bin/bash", "/bin/bash", Bash},
		{"bash from /usr/bin/bash", "/usr/bin/bash", Bash},
		{"zsh from /bin/zsh", "/bin/zsh", Zsh},
		{"zsh from /usr/bin/zsh", "/usr/bin/zsh", Zsh},
		{"fish from /usr/bin/fish", "/usr/bin/fish", Fish},
		{"pwsh from path", "/usr/local/bin/pwsh", PowerShell},
		{"pwsh.exe from path", "/usr/local/bin/pwsh.exe", PowerShell},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SHELL", tt.envShell)
			got, err := DetectShell("")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestDetectShellUnsupported(t *testing.T) {
	t.Setenv("SHELL", "/bin/csh")
	_, err := DetectShell("")
	assert.ErrorContains(t, err, "unsupported shell")
	assert.ErrorContains(t, err, "supported shells are")
}

func TestDetectShellPowershellExeNonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-windows test")
	}
	// On non-Windows, powershell.exe in $SHELL is unrecognized and should error.
	t.Setenv("SHELL", "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe")
	_, err := DetectShell("")
	assert.ErrorContains(t, err, "unsupported shell")
}

func TestDetectShellPowershellExeWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only test")
	}
	t.Setenv("SHELL", `C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe`)
	got, err := DetectShell("")
	require.NoError(t, err)
	assert.Equal(t, PowerShell5, got)
}

func TestDetectShellEmptyOnUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}
	t.Setenv("SHELL", "")
	_, err := DetectShell("")
	assert.ErrorContains(t, err, "$SHELL is not set")
}

func TestDetectShellFlagOverride(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")

	got, err := DetectShell("bash")
	require.NoError(t, err)
	assert.Equal(t, Bash, got)
}

func TestDetectShellFlagInvalid(t *testing.T) {
	_, err := DetectShell("tcsh")
	assert.ErrorContains(t, err, "unsupported shell")
}

func TestDetectShellFlagPowerShell5NonWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("non-windows test")
	}
	_, err := DetectShell("powershell5")
	assert.ErrorContains(t, err, "only supported on Windows")
}

func TestTargetFilePathBashDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin-only test")
	}

	home := t.TempDir()
	// Neither file exists — should return primary (bash_profile on darwin).
	got := TargetFilePath(Bash, home)
	assert.Equal(t, filepath.Join(home, ".bash_profile"), got)

	// Create .bashrc — should fall back to it since .bash_profile doesn't exist.
	require.NoError(t, os.WriteFile(filepath.Join(home, ".bashrc"), nil, 0o644))
	got = TargetFilePath(Bash, home)
	assert.Equal(t, filepath.Join(home, ".bashrc"), got)

	// Create .bash_profile — should prefer it.
	require.NoError(t, os.WriteFile(filepath.Join(home, ".bash_profile"), nil, 0o644))
	got = TargetFilePath(Bash, home)
	assert.Equal(t, filepath.Join(home, ".bash_profile"), got)
}

func TestTargetFilePathBashLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux-only test")
	}

	home := t.TempDir()
	got := TargetFilePath(Bash, home)
	assert.Equal(t, filepath.Join(home, ".bashrc"), got)
}

func TestTargetFilePathZsh(t *testing.T) {
	home := t.TempDir()
	got := TargetFilePath(Zsh, home)
	assert.Equal(t, filepath.Join(home, ".zshrc"), got)
}

func TestTargetFilePathFish(t *testing.T) {
	home := t.TempDir()
	got := TargetFilePath(Fish, home)
	assert.Equal(t, filepath.Join(home, ".config", "fish", "completions", "databricks.fish"), got)
}

func TestTargetFilePathPowerShellUnix(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-only test")
	}
	home := t.TempDir()
	got := TargetFilePath(PowerShell, home)
	assert.Equal(t, filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"), got)
}

func TestTargetFilePathPowerShell5(t *testing.T) {
	home := t.TempDir()
	got := TargetFilePath(PowerShell5, home)
	assert.Equal(t, filepath.Join(home, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"), got)
}

func TestShimContent(t *testing.T) {
	tests := []struct {
		shell    Shell
		contains string
	}{
		{Bash, `eval "$(databricks completion bash)"`},
		{Zsh, `eval "$(databricks completion zsh)"`},
		{Fish, "databricks completion fish | source"},
		{PowerShell, "databricks completion powershell | Out-String | Invoke-Expression"},
		{PowerShell5, "databricks completion powershell | Out-String | Invoke-Expression"},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			content := ShimContent(tt.shell)
			assert.Contains(t, content, BeginMarker)
			assert.Contains(t, content, EndMarker)
			assert.Contains(t, content, tt.contains)
		})
	}
}

func TestDisplayName(t *testing.T) {
	tests := []struct {
		shell    Shell
		expected string
	}{
		{Bash, "bash"},
		{Zsh, "zsh"},
		{Fish, "fish"},
		{PowerShell, "powershell (pwsh 7+)"},
		{PowerShell5, "powershell5 (Windows PowerShell 5.1)"},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.shell.DisplayName())
		})
	}
}
