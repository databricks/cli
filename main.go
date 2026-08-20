package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/cmd"
	"github.com/databricks/cli/cmd/root"

	// Registers a disk-cached HostMetadataResolver factory on the SDK so every
	// *config.Config the CLI constructs reuses the cached /.well-known lookup.
	_ "github.com/databricks/cli/libs/hostmetadata"
)

// commandArgs switches copied Windows helpers into Docker get mode and rejects every other helper operation.
// See https://docs.docker.com/reference/cli/docker/login/#credential-helper-protocol.
func commandArgs(executable string, args []string) ([]string, bool, error) {
	// Windows installs a copy of this binary as the helper, so argv[0] selects credential-helper mode.
	base := executable
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}
	helperName := "docker-credential-databricks"
	if !strings.EqualFold(base, helperName) && !strings.EqualFold(base, helperName+".exe") {
		return args, false, nil
	}
	if len(args) != 1 || args[0] != "get" {
		return nil, true, errors.New("docker-credential-databricks only supports get")
	}
	return []string{"auth", "token", "--format=docker"}, true, nil
}

func main() {
	args, dockerHelper, err := commandArgs(os.Args[0], os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if dockerHelper {
		_ = os.Setenv("DATABRICKS_LOG_FILE", "stderr")
	}

	// Configure DATABRICKS_CLI_PATH only if our caller intends to use this specific version of this binary.
	// Otherwise, if it is equal to its basename, processes can find it in $PATH.
	// This runs in main rather than in a package init so that importing CLI
	// packages (e.g. from test binaries or generators) does not mutate the
	// process environment.
	arg0 := os.Args[0]
	if !dockerHelper && arg0 != filepath.Base(arg0) {
		os.Setenv("DATABRICKS_CLI_PATH", arg0)
	}

	ctx := context.Background()
	cli := cmd.New(ctx)
	cli.SetArgs(args)
	err = root.Execute(ctx, cli)
	if err != nil {
		os.Exit(1)
	}
}
