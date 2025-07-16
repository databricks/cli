package profile

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
)

type FileProfilerImpl struct{}

func (f FileProfilerImpl) getPath(ctx context.Context, replaceHomeDirWithTilde bool) (string, error) {
	configFile := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	if configFile == "" {
		configFile = "~/.databrickscfg"
	}
	if !replaceHomeDirWithTilde {
		return configFile, nil
	}
	homedir, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", err
	}
	configFile = strings.Replace(configFile, homedir, "~", 1)
	return configFile, nil
}

// Get the path to the .databrickscfg file, falling back to the default in the current user's home directory.
func (f FileProfilerImpl) GetPath(ctx context.Context) (string, error) {
	fp, err := f.getPath(ctx, true)
	if err != nil {
		return "", err
	}
	return filepath.Clean(fp), nil
}

var ErrNoConfiguration = errors.New("no configuration file found")

func (f FileProfilerImpl) Get(ctx context.Context) (*config.File, error) {
	path, err := f.getPath(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("cannot determine Databricks config file path: %w", err)
	}
	if strings.HasPrefix(path, "~") {
		homedir, err := env.UserHomeDir(ctx)
		if err != nil {
			return nil, err
		}
		path = filepath.Join(homedir, path[1:])
	}
	configFile, err := config.LoadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// downstreams depend on ErrNoConfiguration. TODO: expose this error through SDK
		return nil, fmt.Errorf("%w at %s; please create one by running 'databricks configure'", ErrNoConfiguration, path)
	} else if err != nil {
		return nil, err
	}
	return configFile, nil
}

func (f FileProfilerImpl) LoadProfiles(ctx context.Context, fn ProfileMatchFunction) (profiles Profiles, err error) {
	file, err := f.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot load Databricks config file: %w", err)
	}

	// Iterate over sections and collect matching profiles.
	for _, v := range file.Sections() {
		all := v.KeysHash()
		host, ok := all["host"]
		if !ok {
			// invalid profile
			continue
		}
		profile := Profile{
			Name:      v.Name(),
			Host:      host,
			AccountID: all["account_id"],
			ClusterID: all["cluster_id"],
		}
		if fn(profile) {
			profiles = append(profiles, profile)
		}
	}

	return
}

func ProfileCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	profiles, err := DefaultProfiler.LoadProfiles(cmd.Context(), MatchAllProfiles)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return profiles.Names(), cobra.ShellCompDirectiveNoFileComp
}
