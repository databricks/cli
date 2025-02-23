package project

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/process"
	"golang.org/x/mod/semver"
)

var ErrNoPythonInterpreters = errors.New("no python3 interpreters found")

const (
	officialMswinPython  = "(Python Official) https://python.org/downloads/windows"
	microsoftStorePython = "(Microsoft Store) https://apps.microsoft.com/store/search?publisher=Python%20Software%20Foundation"
)

const worldWriteable = 0o002

type Interpreter struct {
	Version string
	Path    string
}

func (i Interpreter) String() string {
	return fmt.Sprintf("%s (%s)", i.Version, i.Path)
}

type allInterpreters []Interpreter

func (a allInterpreters) Latest() Interpreter {
	return a[len(a)-1]
}

func (a allInterpreters) AtLeast(minimalVersion string) (*Interpreter, error) {
	canonicalMinimalVersion := semver.Canonical("v" + strings.TrimPrefix(minimalVersion, "v"))
	if canonicalMinimalVersion == "" {
		return nil, fmt.Errorf("invalid SemVer: %s", minimalVersion)
	}
	for _, interpreter := range a {
		cmp := semver.Compare(interpreter.Version, canonicalMinimalVersion)
		if cmp < 0 {
			continue
		}
		return &interpreter, nil
	}
	return nil, fmt.Errorf("cannot find Python greater or equal to %s", canonicalMinimalVersion)
}

func DetectInterpreters(ctx context.Context) (allInterpreters, error) {
	found := allInterpreters{}
	seen := map[string]bool{}
	executables, err := pythonicExecutablesFromPathEnvironment(ctx)
	if err != nil {
		return nil, err
	}
	log.Debugf(ctx, "found %d potential alternative Python versions in $PATH", len(executables))
	for _, resolved := range executables {
		if seen[resolved] {
			continue
		}
		seen[resolved] = true
		// probe the binary version by executing it, like `python --version`
		// and parsing the output.
		//
		// Keep in mind, that mswin installations get python.exe and pythonw.exe,
		// which are slightly different: see https://stackoverflow.com/a/30313091
		out, err := process.Background(ctx, []string{resolved, "--version"})
		var processErr *process.ProcessError
		if errors.As(err, &processErr) {
			log.Debugf(ctx, "failed to check version for %s: %s", resolved, processErr.Err)
			continue
		}
		if err != nil {
			log.Debugf(ctx, "failed to check version for %s: %s", resolved, err)
			continue
		}
		version := validPythonVersion(ctx, resolved, out)
		if version == "" {
			continue
		}
		found = append(found, Interpreter{
			Version: version,
			Path:    resolved,
		})
	}
	if runtime.GOOS == "windows" && len(found) == 0 {
		return nil, fmt.Errorf("%w. Install them from %s or %s and restart the shell",
			ErrNoPythonInterpreters, officialMswinPython, microsoftStorePython)
	}
	if len(found) == 0 {
		return nil, ErrNoPythonInterpreters
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

func pythonicExecutablesFromPathEnvironment(ctx context.Context) (out []string, err error) {
	paths := strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))
	for _, prefix := range paths {
		info, err := os.Stat(prefix)
		if errors.Is(err, fs.ErrNotExist) {
			// some directories in $PATH may not exist
			continue
		}
		if errors.Is(err, fs.ErrPermission) {
			// some directories we cannot list
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", prefix, err)
		}
		if !info.IsDir() {
			continue
		}
		perm := info.Mode().Perm()
		if runtime.GOOS != "windows" && perm&worldWriteable != 0 {
			// we try not to run any python binary that sits in a writable folder by all users.
			// this is mainly to avoid breaking the security model on a multi-user system.
			// If the PATH is pointing somewhere untrusted it is the user fault, but we can
			// help here.
			//
			// See https://github.com/databricks/cli/pull/805#issuecomment-1735403952
			log.Debugf(ctx, "%s is world-writeable (%s), skipping for security reasons", prefix, perm)
			continue
		}
		entries, err := os.ReadDir(prefix)
		if errors.Is(err, fs.ErrPermission) {
			// some directories we cannot list
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
			// If Python3 is installed on Windows through GUI installer app that was
			// downloaded from https://python.org/downloads/windows, it may appear
			// in $PATH as `python`, even though it means Python 2.7 in all other
			// operating systems (macOS, Linux).
			//
			// See https://github.com/databrickslabs/ucx/issues/281
			if !strings.HasPrefix(v.Name(), "python") {
				continue
			}
			bin := filepath.Join(prefix, v.Name())
			resolved, err := filepath.EvalSymlinks(bin)
			if err != nil {
				log.Debugf(ctx, "cannot resolve symlink for %s: %s", bin, resolved)
				continue
			}
			out = append(out, resolved)
		}
	}
	return out, nil
}

func validPythonVersion(ctx context.Context, resolved, out string) string {
	out = strings.TrimSpace(out)
	log.Debugf(ctx, "%s --version: %s", resolved, out)

	words := strings.Split(out, " ")
	// The Python distribution from the Windows Store is available in $PATH as `python.exe`
	// and `python3.exe`, even though it symlinks to a real file packaged with some versions of Windows:
	// /c/Program Files/WindowsApps/Microsoft.DesktopAppInstaller_.../AppInstallerPythonRedirector.exe.
	// Executing the `python` command from this distribution opens the Windows Store, allowing users to
	// download and install Python. Once installed, it replaces the `python.exe` and `python3.exe`` stub
	// with the genuine Python executable. Additionally, once user installs from the main installer at
	// https://python.org/downloads/windows, it does not replace this stub.
	//
	// However, a drawback is that if this initial stub is run with any command line arguments, it quietly
	// fails to execute. According to https://github.com/databrickslabs/ucx/issues/281, it can be
	// detected by seeing just the "Python" output without any version info from the `python --version`
	// command execution.
	//
	// See https://github.com/pypa/packaging-problems/issues/379
	// See https://bugs.python.org/issue41327
	if len(words) < 2 {
		log.Debugf(ctx, "%s --version: stub from Windows Store", resolved)
		return ""
	}

	if words[0] != "Python" {
		log.Debugf(ctx, "%s --version: not a Python", resolved)
		return ""
	}

	lastWord := words[len(words)-1]
	version := semver.Canonical("v" + lastWord)
	if version == "" {
		log.Debugf(ctx, "%s --version: invalid SemVer: %s", resolved, lastWord)
		return ""
	}

	return version
}
