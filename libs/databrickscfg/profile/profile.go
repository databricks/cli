package profile

import (
	"strings"

	"github.com/databricks/databricks-sdk-go/config"
)

// Profile holds a subset of the keys in a databrickscfg profile.
// It should only be used for prompting and filtering.
// Use its name to construct a config.Config.
type Profile struct {
	Name                 string
	Host                 string
	AccountID            string
	WorkspaceID          string
	ClusterID            string
	ServerlessComputeID  string
	HasClientCredentials bool
	Scopes               string
	AuthType             string
}

// FromConfig returns the simplified profile represented by a resolved config.
func FromConfig(cfg *config.Config) Profile {
	return Profile{
		Name:                 cfg.Profile,
		Host:                 cfg.Host,
		AccountID:            cfg.AccountID,
		WorkspaceID:          cfg.WorkspaceID,
		ClusterID:            cfg.ClusterID,
		ServerlessComputeID:  cfg.ServerlessComputeID,
		HasClientCredentials: cfg.ClientID != "" && cfg.ClientSecret != "",
		Scopes:               strings.Join(cfg.Scopes, ","),
		AuthType:             cfg.AuthType,
	}
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

// SearchCaseInsensitive matches the cmdio.SelectOptions.Searcher signature so
// the user can immediately start typing to narrow down the list.
func (p Profiles) SearchCaseInsensitive(input string, index int) bool {
	input = strings.ToLower(input)
	name := strings.ToLower(p[index].Name)
	host := strings.ToLower(p[index].Host)
	accountID := strings.ToLower(p[index].AccountID)
	return strings.Contains(name, input) || strings.Contains(host, input) || strings.Contains(accountID, input)
}

func (p Profiles) Names() []string {
	names := make([]string, len(p))
	for i, v := range p {
		names[i] = v.Name
	}
	return names
}
