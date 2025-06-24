package profile

import (
	"strings"

	"github.com/databricks/databricks-sdk-go/config"
)

// Profile holds a subset of the keys in a databrickscfg profile.
// It should only be used for prompting and filtering.
// Use its name to construct a config.Config.
type Profile struct {
	Name      string
	Host      string
	AccountID string
	ClusterID string
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

// SearchCaseInsensitive implements the promptui.Searcher interface.
// This allows the user to immediately starting typing to narrow down the list.
func (p Profiles) SearchCaseInsensitive(input string, index int) bool {
	input = strings.ToLower(input)
	name := strings.ToLower(p[index].Name)
	host := strings.ToLower(p[index].Host)
	return strings.Contains(name, input) || strings.Contains(host, input)
}

func (p Profiles) Names() []string {
	names := make([]string, len(p))
	for i, v := range p {
		names[i] = v.Name
	}
	return names
}
