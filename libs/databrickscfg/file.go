package databrickscfg

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/ini.v1"
)

// File represents the contents of a databrickscfg file.
type File struct {
	*ini.File

	path string
}

// Path returns the path of the loaded databrickscfg file.
func (f *File) Path() string {
	return f.path
}

// LoadFile loads the databrickscfg file at the specified path.
// The function loads ~/.databrickscfg if the specified path is an empty string.
// The function expands ~ to the user's home directory.
func LoadFile(path string) (*File, error) {
	if path == "" {
		path = "~/.databrickscfg"
	}

	// Expand ~ to home directory.
	if strings.HasPrefix(path, "~") {
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot find homedir: %w", err)
		}
		path = fmt.Sprintf("%s%s", homedir, path[1:])
	}

	iniFile, err := ini.Load(path)
	if err != nil {
		return nil, err
	}

	return &File{iniFile, path}, err
}
