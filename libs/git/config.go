package git

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
type Config map[string]string

var regexpComment = regexp.MustCompile(`[#;].*$`)
var regexpSection = regexp.MustCompile(`^\[([\w\-\.]+)(\s+"(.*)")?\]$`)
var regexpVariable = regexp.MustCompile(`^(\w[\w-]*)(\s*=\s*(.*))?$`)

func (c Config) load(r io.Reader) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	prefix := []string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		// Below implements https://git-scm.com/docs/git-config#_syntax.
		line := scanner.Text()

		// Remove comments and trim whitespace.
		line = regexpComment.ReplaceAllString(line, "")
		line = strings.TrimSpace(line)

		// Ignore empty lines.
		if len(line) == 0 {
			continue
		}

		// See if this line defines a new section or subsection.
		match := regexpSection.FindStringSubmatch(line)
		if match != nil {
			// Section names are case-insensitive.
			sectionName := strings.ToLower(match[1])
			// Subsection names are case sensitive.
			subsectionName := match[3]

			// Similar to `git config --global --list` we use the
			// section and subsection name as prefix for variables.
			prefix = []string{sectionName}
			if subsectionName != "" {
				prefix = append(prefix, subsectionName)
			}

			continue
		}

		// See if this line defines a variable.
		match = regexpVariable.FindStringSubmatch(line)
		if match != nil && len(prefix) > 0 {
			var key, value string

			key = strings.ToLower(match[1])
			if match[2] == "" {
				value = "true"
			} else {
				value = match[3]
			}

			// Expand ~/ to home directory.
			if strings.HasPrefix(value, "~/") {
				value = filepath.Join(home, value[2:])
			}

			c[strings.Join(append(prefix, key), ".")] = value
		}
	}

	if scanner.Err() != nil {
		return scanner.Err()
	}

	return nil
}

func (c Config) loadFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		// From https://git-scm.com/docs/git-config#FILES:
		//
		// > If the global or the system-wide configuration files
		// > are missing or unreadable they will be ignored.
		return nil
	}

	defer f.Close()
	return c.load(f)
}

func globalGitConfig(home string) (config Config, err error) {
	xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfigHome == "" {
		xdgConfigHome = filepath.Join(home, ".config")
	}

	config = make(Config)
	err = config.loadFile(filepath.Join(xdgConfigHome, "git/config"))
	if err != nil {
		return nil, err
	}
	err = config.loadFile(filepath.Join(home, ".gitconfig"))
	if err != nil {
		return nil, err
	}

	return config, nil
}

func coreExcludesFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	config, err := globalGitConfig(home)
	if err != nil {
		return "", err
	}

	path := config["core.excludesfile"]
	if path == "" {
		// Defaults to $XDG_CONFIG_HOME/git/ignore.
		xdgConfigHome := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfigHome == "" {
			// If $XDG_CONFIG_HOME is either not set or empty,
			// $HOME/.config/git/ignore is used instead.
			xdgConfigHome = filepath.Join(home, ".config")
		}

		path = filepath.Join(xdgConfigHome, "git/ignore")
	}

	// Only return if this file is stat-able or doesn't exist (yet).
	// If there are other problems accessing this file we would
	// run into them at a later point anyway.
	_, err = os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	return path, nil
}
