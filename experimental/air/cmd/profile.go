package aircmd

import (
	"errors"
	"io/fs"

	"github.com/databricks/cli/libs/databrickscfg"
	"github.com/databricks/databricks-sdk-go/config"
)

// profileValue returns a property from the selected configuration profile. An
// empty profile name uses the configured default-profile resolution order.
func profileValue(cfg *config.Config, key string) (string, error) {
	file, err := config.LoadFile(cfg.ConfigFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	profile := cfg.Profile
	if profile == "" {
		profile = databrickscfg.GetDefaultProfileFrom(file)
	}
	if profile == "" {
		return "", nil
	}
	if !file.HasSection(profile) {
		return "", nil
	}
	section := file.Section(profile)
	if !section.HasKey(key) {
		return "", nil
	}
	return section.Key(key).Value(), nil
}
