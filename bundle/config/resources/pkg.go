package resources

import "path/filepath"

type Paths struct {
	ConfigFilePath string `json:"-" bundle:"readonly"`
}

func (p *Paths) ConfigFileDirectory() string {
	if p.ConfigFilePath == "" {
		return "."
	}
	return filepath.Dir(p.ConfigFilePath)
}
