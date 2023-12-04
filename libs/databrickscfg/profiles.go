package databrickscfg

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

// Profile holds a subset of the keys in a databrickscfg profile.
// It should only be used for prompting and filtering.
// Use its name to construct a config.Config.
type Profile struct {
	Name      string
	Host      string
	AccountID string
}

func (p Profile) Cloud() string {
	cfg := config.Config{Host: p.Host}
	switch {
	case cfg.IsAws():
		return "AWS"
	case cfg.IsAzure():
		return "Azure"
	case cfg.IsGcp():
		return "GCP"
	default:
		return ""
	}
}

type Profiles []Profile

func (p Profiles) Names() []string {
	names := make([]string, len(p))
	for i, v := range p {
		names[i] = v.Name
	}
	return names
}

// SearchCaseInsensitive implements the promptui.Searcher interface.
// This allows the user to immediately starting typing to narrow down the list.
func (p Profiles) SearchCaseInsensitive(input string, index int) bool {
	input = strings.ToLower(input)
	name := strings.ToLower(p[index].Name)
	host := strings.ToLower(p[index].Host)
	return strings.Contains(name, input) || strings.Contains(host, input)
}

type ProfileMatchFunction func(Profile) bool

func MatchWorkspaceProfiles(p Profile) bool {
	return p.AccountID == ""
}

func MatchAccountProfiles(p Profile) bool {
	return p.Host != "" && p.AccountID != ""
}

func MatchAllProfiles(p Profile) bool {
	return true
}

// Get the path to the .databrickscfg file, falling back to the default in the current user's home directory.
func GetPath(ctx context.Context) (string, error) {
	configFile := env.Get(ctx, "DATABRICKS_CONFIG_FILE")
	if configFile == "" {
		configFile = "~/.databrickscfg"
	}
	if strings.HasPrefix(configFile, "~") {
		homedir, err := env.UserHomeDir(ctx)
		if err != nil {
			return "", err
		}
		configFile = filepath.Join(homedir, configFile[1:])
	}
	return configFile, nil
}

var ErrNoConfiguration = errors.New("no configuration file found")

func Get(ctx context.Context) (*config.File, error) {
	path, err := GetPath(ctx)
	if err != nil {
		return nil, fmt.Errorf("cannot determine Databricks config file path: %w", err)
	}
	configFile, err := config.LoadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// downstreams depend on ErrNoConfiguration. TODO: expose this error through SDK
		return nil, fmt.Errorf("%w at %s; please create one first", ErrNoConfiguration, path)
	} else if err != nil {
		return nil, err
	}
	return configFile, nil
}

func LoadProfiles(ctx context.Context, fn ProfileMatchFunction) (file string, profiles Profiles, err error) {
	f, err := Get(ctx)
	if err != nil {
		return "", nil, fmt.Errorf("cannot load Databricks config file: %w", err)
	}

	// Replace homedir with ~ if applicable.
	// This is to make the output more readable.
	file = filepath.Clean(f.Path())
	home, err := env.UserHomeDir(ctx)
	if err != nil {
		return "", nil, err
	}
	homedir := filepath.Clean(home)
	if strings.HasPrefix(file, homedir) {
		file = "~" + file[len(homedir):]
	}

	// Iterate over sections and collect matching profiles.
	for _, v := range f.Sections() {
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
		}
		if fn(profile) {
			profiles = append(profiles, profile)
		}
	}

	return
}

func ProfileCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	_, profiles, err := LoadProfiles(cmd.Context(), MatchAllProfiles)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return profiles.Names(), cobra.ShellCompDirectiveNoFileComp
}
