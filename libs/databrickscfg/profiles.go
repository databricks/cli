package databrickscfg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
func GetDatabricksCfgPath() (string, error) {
	configFile := os.Getenv("DATABRICKS_CONFIG_FILE")
	if configFile == "" {
		configFile = "~/.databrickscfg"
	}
	if strings.HasPrefix(configFile, "~") {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("cannot find homedir: %w", err)
		}
		configFile = filepath.Join(homedir, configFile[1:])
	}
	return configFile, nil
}

func GetDatabricksCfg() (*config.File, error) {
	configFile, err := GetDatabricksCfgPath()
	if err != nil {
		return nil, fmt.Errorf("cannot determine Databricks config file path: %w", err)
	}
	return config.LoadFile(configFile)
}

func LoadProfiles(fn ProfileMatchFunction) (file string, profiles Profiles, err error) {
	f, err := GetDatabricksCfg()
	if err != nil {
		return "", nil, fmt.Errorf("cannot load Databricks config file: %w", err)
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Replace homedir with ~ if applicable.
	// This is to make the output more readable.
	file = f.Path()
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
	_, profiles, err := LoadProfiles(MatchAllProfiles)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	return profiles.Names(), cobra.ShellCompDirectiveNoFileComp
}
