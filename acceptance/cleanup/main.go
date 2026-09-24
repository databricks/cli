// Command cleanup implements a standalone bundle cleanup program. It is invoked as a
// separate always()-triggered workflow job so it runs even when a test job times out or
// is cancelled — unlike a t.Cleanup, which go test skips in those cases.
package main

import (
	"context"
	"errors"
	"flag"

	"github.com/databricks/cli/acceptance/internal/bundlecleanup"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
)

func main() {
	ctx := context.Background()
	var cliPath string
	var suppliedPrefix string
	flag.StringVar(&cliPath, "cli", "", "path to databricks CLI binary (required)")
	flag.StringVar(&suppliedPrefix, "prefix", "", "exact local cleanup prefix; defaults to the current CI run prefix")
	flag.Parse()

	if cliPath == "" {
		log.Errorf(ctx, "-cli: path to databricks CLI binary is required")
	}
	prefix, err := resolvePrefix(ctx, suppliedPrefix)
	if err != nil {
		log.Errorf(ctx, "%s", err)
		return
	}
	log.Infof(ctx, "bundle cleanup prefix: %q", prefix)
	if err := bundlecleanup.CleanBundles(ctx, cliPath, prefix); err != nil {
		log.Errorf(ctx, "failed to clean bundles: %s", err)
	}
}

func resolvePrefix(ctx context.Context, supplied string) (string, error) {
	if supplied != "" {
		if err := bundlecleanup.ValidatePrefix(supplied, env.Get(ctx, "GITHUB_RUN_ID")); err != nil {
			return "", err
		}
		return supplied, nil
	}
	prefix, err := bundlecleanup.CleanPrefix(ctx)
	if err != nil {
		return "", err
	}
	if prefix == "" {
		return "", errors.New("-prefix is required outside CI; use the exact local prefix printed by the acceptance run")
	}
	return prefix, nil
}
