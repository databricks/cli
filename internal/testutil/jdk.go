package testutil

import (
	"context"
	"os/exec"
	"strings"
)

// HasJDK checks if the specified Java version is available in the system.
// It returns true if the required JDK version is present, false otherwise.
// This is a non-failing variant of RequireJDK.
//
// Example output of `java -version` in eclipse-temurin:8:
// openjdk version "1.8.0_442"
// OpenJDK Runtime Environment (Temurin)(build 1.8.0_442-b06)
// OpenJDK 64-Bit Server VM (Temurin)(build 25.442-b06, mixed mode)
//
// Example output of `java -version` in java11 (homebrew):
// openjdk version "11.0.26" 2025-01-21
// OpenJDK Runtime Environment Homebrew (build 11.0.26+0)
// OpenJDK 64-Bit Server VM Homebrew (build 11.0.26+0, mixed mode)
func HasJDK(t TestingT, ctx context.Context, version string) bool {
	t.Helper()

	// Try to execute "java -version" command
	cmd := exec.CommandContext(ctx, "java", "-version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Failed to execute java -version: %v", err)
		return false
	}

	javaVersionOutput := string(output)

	// Check if the output contains the expected version
	expectedVersionString := "version \"" + version
	if strings.Contains(javaVersionOutput, expectedVersionString) {
		t.Logf("Detected JDK version %s", version)
		return true
	}

	t.Logf("Required JDK version %s not found, instead got: %s", version, javaVersionOutput)
	return false
}
