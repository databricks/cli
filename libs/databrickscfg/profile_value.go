package databrickscfg

import (
	"errors"
	"io/fs"

	"github.com/databricks/databricks-sdk-go/config"
)

// ProfileValue returns a property from the selected configuration profile. An
// empty profile name uses the configured default-profile resolution order.
func ProfileValue(cfg *config.Config, key string) (string, error) {
	file, err := config.LoadFile(cfg.ConfigFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	profile := cfg.Profile
	if profile == "" {
		profile = GetDefaultProfileFrom(file)
	}
	if profile == "" {
		return "", nil
	}
	section, err := file.GetSection(profile)
	if err != nil {
		return "", nil
	}
	value, err := section.GetKey(key)
	if err != nil {
		return "", nil
	}
	return value.Value(), nil
}
