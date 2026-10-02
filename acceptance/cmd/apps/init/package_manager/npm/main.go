package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

// main emulates npm without a local Node.js installation or package downloads.
func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// run validates the command and the package.json visible to the installer.
//
//nolint:forbidigo // This standalone acceptance helper cannot import the CLI's libs/env package.
func run() error {
	version := os.Getenv("CLI_TEST_NPM_VERSION")
	if version == "unavailable" {
		return errors.New("npm is unavailable")
	}
	switch {
	case slices.Equal(os.Args[1:], []string{"--version"}):
		if os.Getenv("COREPACK_ENABLE_PROJECT_SPEC") != "0" {
			return errors.New("version detection must ignore inherited Corepack pins")
		}
		fmt.Println(version)
		return nil
	case slices.Equal(os.Args[1:], []string{"ci", "--include=dev", "--no-audit", "--no-fund", "--prefer-offline"}):
		data, err := os.ReadFile("package.json")
		if err != nil {
			return err
		}
		var pkg struct {
			PackageManager string `json:"packageManager"`
		}
		if err := json.Unmarshal(data, &pkg); err != nil {
			return err
		}
		if pkg.PackageManager != "npm@"+version {
			return fmt.Errorf("install saw packageManager %q, expected npm@%s", pkg.PackageManager, version)
		}
		if err := os.MkdirAll("node_modules", 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join("node_modules", "installed-pin"), []byte(pkg.PackageManager+"\n"), 0o644)
	default:
		return fmt.Errorf("unexpected npm arguments: %q", os.Args[1:])
	}
}
