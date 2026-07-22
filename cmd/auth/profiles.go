package auth

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/spf13/cobra"
	"gopkg.in/ini.v1"
)

type profileMetadata struct {
	Name        string `json:"name"`
	Host        string `json:"host,omitempty"`
	AccountID   string `json:"account_id,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	Cloud       string `json:"cloud"`
	AuthType    string `json:"auth_type"`
	Default     bool   `json:"default,omitempty"`
}

func (c *profileMetadata) IsEmpty() bool {
	return c.Host == "" && c.AccountID == ""
}

// cloudFromHost classifies a host into a cloud from its pattern alone, without
// resolving the config or making a network call.
func cloudFromHost(host string) string {
	cfg := config.Config{Host: host}
	switch {
	case cfg.IsAws():
		return "aws"
	case cfg.IsAzure():
		return "azure"
	case cfg.IsGcp():
		return "gcp"
	}
	return ""
}

func newProfilesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "Lists profiles from ~/.databrickscfg",
		Long: `Lists the profiles configured in ~/.databrickscfg. This is a fast,
offline listing that only reads the config file. To check whether a profile can
authenticate, run "databricks auth describe".`,
		Annotations: map[string]string{
			"template": cmdio.Heredoc(`
			{{header "Name"}}	{{header "Host"}}
			{{range .Profiles}}{{.Name | green}}{{if .Default}} (Default){{end}}	{{.Host|cyan}}
			{{end}}`),
		},
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		var profiles []*profileMetadata
		iniFile, err := profile.DefaultProfiler.Get(cmd.Context())
		if errors.Is(err, fs.ErrNotExist) {
			// return empty list for non-configured machines
			iniFile = &config.File{
				File: &ini.File{},
			}
		} else if err != nil {
			return fmt.Errorf("cannot parse config file: %w", err)
		}

		defaultProfile := databrickscfg.GetConfiguredDefaultProfileFrom(iniFile)

		for _, v := range iniFile.Sections() {
			hash := v.KeysHash()
			profile := &profileMetadata{
				Name:        v.Name(),
				Host:        hash["host"],
				AccountID:   hash["account_id"],
				WorkspaceID: hash["workspace_id"],
				AuthType:    hash["auth_type"],
				Default:     v.Name() == defaultProfile,
			}
			if profile.IsEmpty() {
				continue
			}
			profile.Cloud = cloudFromHost(profile.Host)
			profiles = append(profiles, profile)
		}
		return cmdio.Render(cmd.Context(), struct {
			Profiles []*profileMetadata `json:"profiles"`
		}{profiles})
	}

	return cmd
}
