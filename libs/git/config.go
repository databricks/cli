package git

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/ini.v1"
)

// Config holds the entries of a gitconfig file.
//
// As map key we join the section name, optionally the subsection name,
// and the variable name with dots. The result is ~equivalent to the
// output of `git config --global --list`.
//
// While this doesn't capture the full richness of gitconfig it's good
// enough for the basic properties we care about (e.g. under `core`).
//
// Also see: https://git-scm.com/docs/git-config.
type config struct {
	home      string
	variables map[string]string
}

func newConfig() (*config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	return &config{
		home:      home,
		variables: make(map[string]string),
	}, nil
}

var regexpSection = regexp.MustCompile(`^([\w\-\.]+)(\s+"(.*)")?$`)

func (c config) walkSections(prefix []string, sections []*ini.Section) error {
	for _, section := range sections {
		prevPrefix := prefix

		// Detect and split section name that includes subsection name.
		if match := regexpSection.FindStringSubmatch(section.Name()); match != nil {
			prefix = append(prefix, match[1])
			if match[3] != "" {
				prefix = append(prefix, match[3])
			}
		} else {
			prefix = append(prefix, section.Name())
		}

		// Add variables in this section.
		for key, value := range section.KeysHash() {
			key = strings.ToLower(key)
			if value == "" {
				value = "true"
			}

			// Expand ~/ to home directory.
			if strings.HasPrefix(value, "~/") {
				value = filepath.Join(c.home, value[2:])
			}

			c.variables[strings.Join(append(prefix, key), ".")] = value
		}

		// Recurse into child sections.
		err := c.walkSections(prefix, section.ChildSections())
		if err != nil {
			return err
		}

		prefix = prevPrefix
	}

	return nil
}

func (c config) load(r io.Reader) error {
	iniFile, err := ini.InsensitiveLoad(r)
	if err != nil {
		return err
	}

	// Collapse sections, subsections, and keys, into a flat namespace.
	err = c.walkSections([]string{}, iniFile.Sections())
	if err != nil {
		return err
	}

	return nil
}

func (c config) loadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		// From https://git-scm.com/docs/git-config#FILES:
		//
		// > If the global or the system-wide configuration files
		// > are missing or unreadable they will be ignored.
		return nil
	}

	defer f.Close()

	err = c.load(f)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", path, err)
	}
	return nil
}

func (c config) coreExcludesFile() (string, error) {
	path := c.variables["core.excludesfile"]
	if path == "" {
		// Defaults to $XDG_CONFIG_HOME/git/ignore.
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome == "" {
			// If $XDG_CONFIG_HOME is either not set or empty,
			// $HOME/.config/git/ignore is used instead.
			xdgConfigHome = filepath.Join(c.home, ".config")
		}

		path = filepath.Join(xdgConfigHome, "git/ignore")
	}

	// Only return if this file is stat-able or doesn't exist (yet).
	// If there are other problems accessing this file we would
	// run into them at a later point anyway.
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	return path, nil
}

func globalGitConfig() (*config, error) {
	config, err := newConfig()
	if err != nil {
		return nil, err
	}
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		xdgConfigHome = filepath.Join(config.home, ".config")
	}
	err = config.loadFile(filepath.Join(xdgConfigHome, "git/config"))
	if err != nil {
		return nil, err
	}
	err = config.loadFile(filepath.Join(config.home, ".gitconfig"))
	if err != nil {
		return nil, err
	}
	return config, nil
}
