package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
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
	switch {
	case slices.Equal(os.Args[1:], []string{"--version"}):
		if version == "unavailable" {
			return errors.New("npm is unavailable")
		}
		if os.Getenv("COREPACK_ENABLE_PROJECT_SPEC") != "0" {
			return errors.New("version detection must ignore inherited Corepack pins")
		}
		fmt.Println(version)
		return nil
	case slices.Equal(os.Args[1:], []string{"ci", "--include=dev", "--no-audit", "--no-fund", "--prefer-offline"}),
		slices.Equal(os.Args[1:], []string{"install", "--frozen-lockfile", "--prod=false"}):
		switch os.Getenv("CLI_TEST_INSTALL_MODE") {
		case "fail":
			return errors.New("install failed")
		case "wait":
			time.Sleep(time.Minute)
			// A canceled installer must never reach this marker outside the scaffold.
			return os.WriteFile(filepath.Join("..", "uncanceled-install"), nil, 0o644)
		case "retry":
			if _, err := os.Stat("install-attempt"); errors.Is(err, os.ErrNotExist) {
				if err := os.WriteFile("install-attempt", nil, 0o644); err != nil {
					return err
				}
				return errors.New("retry in foreground")
			}
		}
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
		wantPin := "npm@" + version
		if strings.HasPrefix(filepath.Base(os.Args[0]), "pnpm") {
			wantPin = "pnpm@10.17.1+sha224.abc"
		}
		if pkg.PackageManager != wantPin {
			return fmt.Errorf("install saw packageManager %q, expected %s", pkg.PackageManager, wantPin)
		}
		if err := os.MkdirAll("node_modules", 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join("node_modules", "installed-pin"), []byte(pkg.PackageManager+"\n"), 0o644)
	case slices.Equal(os.Args[1:], []string{"appkit", "setup", "--write"}):
		return errors.New("setup failed")
	case slices.Equal(os.Args[1:], []string{"run", "dev"}):
		return errors.New("dev failed")
	default:
		return fmt.Errorf("unexpected npm arguments: %q", os.Args[1:])
	}
}
