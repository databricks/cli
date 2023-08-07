package python

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
	"golang.org/x/mod/semver"
)

type Interpreter struct {
	Version string
	Binary  string
}

func (i Interpreter) String() string {
	return fmt.Sprintf("%s (%s)", i.Version, i.Binary)
}

type AllInterpreters []Interpreter

func (a AllInterpreters) Latest() Interpreter {
	return a[len(a)-1]
}

func DetectInterpreters(ctx context.Context) (AllInterpreters, error) {
	found := AllInterpreters{}
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	seen := map[string]bool{}
	for _, prefix := range paths {
		entries, err := os.ReadDir(prefix)
		if os.IsNotExist(err) {
			// some directories in $PATH may not exist
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("listing %s: %w", prefix, err)
		}
		for _, v := range entries {
			if v.IsDir() {
				continue
			}
			if strings.Contains(v.Name(), "-") {
				// skip python3-config, python3.10-config, etc
				continue
			}
			if !strings.HasPrefix(v.Name(), "python3") {
				continue
			}
			bin := filepath.Join(prefix, v.Name())
			resolved, err := filepath.EvalSymlinks(bin)
			if err != nil {
				log.Debugf(ctx, "cannot resolve symlink for %s: %s", bin, resolved)
				continue
			}
			if seen[resolved] {
				continue
			}
			seen[resolved] = true
			out, err := process.Background(ctx, []string{resolved, "--version"})
			if err != nil {
				// TODO: skip-and-log or return?
				return nil, err
			}
			words := strings.Split(out, " ")
			if words[0] != "Python" {
				continue
			}
			lastWord := words[len(words)-1]
			version := semver.Canonical("v" + lastWord)
			found = append(found, Interpreter{
				Version: version,
				Binary:  bin,
			})
		}
	}
	if len(found) == 0 {
		return nil, fmt.Errorf("no python3 executables found")
	}
	sort.Slice(found, func(i, j int) bool {
		a := found[i].Version
		b := found[j].Version
		cmp := semver.Compare(a, b)
		if cmp != 0 {
			return cmp < 0
		}
		return a < b
	})
	return found, nil
}
