package py

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/slices"

	"github.com/databricks/bricks/lib/dbr"
	"github.com/databricks/bricks/lib/flavor"
	"github.com/databricks/bricks/lib/spawn"
	"github.com/databricks/bricks/python"
	"github.com/databricks/databricks-sdk-go/databricks/apierr"
	"github.com/databricks/databricks-sdk-go/service/commands"
	"github.com/databricks/databricks-sdk-go/service/libraries"
)

type SetupDotPy struct {
	SetupPy         string `json:"setup_py,omitempty"`
	MirrorLibraries bool   `json:"mirror_libraries,omitempty"`

	venv      string
	wheelName string
}

func (s *SetupDotPy) RequiresCluster() bool {
	return true
}

// Python libraries do not require a restart
func (s *SetupDotPy) RequiresRestart() bool {
	return false
}

func (s *SetupDotPy) setupPyLoc(prj flavor.Project) string {
	if s.SetupPy == "" {
		s.SetupPy = "setup.py"
	}
	return filepath.Join(prj.Root(), s.SetupPy)
}

// We detect only setuptools build backend for now. Hatchling, PDM,
// and Flit _might_ be added in some distant future.
//
// See: https://packaging.python.org/en/latest/tutorials/packaging-projects/
func (s *SetupDotPy) Detected(prj flavor.Project) bool {
	_, err := os.Stat(s.setupPyLoc(prj))
	return err == nil
}

// readDistribution "parses" metadata from setup.py file from context
// of current project root and virtual env
func (s *SetupDotPy) readDistribution(ctx context.Context, prj flavor.Project) (d Distribution, err error) {
	ctx = spawn.WithRoot(ctx, filepath.Dir(s.setupPyLoc(prj)))
	out, err := python.Py(ctx, "-c", commands.TrimLeadingWhitespace(`
		import setuptools, json, sys
		setup_config = {} # actual args for setuptools.dist.Distribution
		def capture(**kwargs): global setup_config; setup_config = kwargs
		setuptools.setup = capture
		import setup
		json.dump(setup_config, sys.stdout)`))
	if err != nil {
		return
	}
	err = json.Unmarshal([]byte(out), &d)
	return
}

func (s *SetupDotPy) Prepare(ctx context.Context, prj flavor.Project, status func(string)) error {
	venv, err := s.fastDetectVirtualEnv(prj.Root())
	if err != nil {
		return err
	}
	s.venv = venv
	if s.venv == "" {
		// this allows CLI to be usable in existing projects with existing virtualenvs
		venv = filepath.Join(prj.Root(), ".databricks") // TODO: integrate with pipenv
		err := os.MkdirAll(venv, 0o700)
		if err != nil {
			return fmt.Errorf("mk venv: %w", err)
		}
		status(fmt.Sprintf("Creating virtualenv in %s", venv))
		_, err = python.Py(ctx, "-m", "venv", venv)
		if err != nil {
			return fmt.Errorf("create venv: %w", err)
		}
		s.venv = venv
		status("Upgrading pip")
		_, err = s.Pip(ctx, "install", "--upgrade", "pip", "wheel")
		if err != nil {
			return fmt.Errorf("upgrade pip: %w", err)
		}
	}
	env, err := s.Freeze(ctx)
	if err != nil {
		return fmt.Errorf("pip freeze: %s", err)
	}
	var remotePkgs []string
	d, err := s.readDistribution(ctx, prj)
	if err != nil {
		return fmt.Errorf("setup.py: %s", err)
	}
	if s.MirrorLibraries {
		// TODO: name `MirrorLibraries` is TBD
		// TODO: must be part of init command survey
		status("Fetching remote libraries")
		remoteInfo, err := s.runtimeInfo(ctx, prj, status)
		if err != nil && errors.As(err, &apierr.APIError{}) {
			return err
		}
		// TODO: check Spark compatibility with locall install_requires
		// TODO: check Python version compatibility with local virtualenv
		if err != errNoCluster {
			skipLibs := []string{
				"dbus-python",
				"distro-info",
				"pip",
				"psycopg2",
				"pygobject",
				"python-apt",
				"requests-unixsocket",
				"setuptools",
				"unattended-upgrades",
				"wheel",
			}
		PYPI:
			for _, pkg := range remoteInfo.PyPI {
				if env.Has(pkg.PyPiName()) {
					continue
				}
				if pkg.Name == d.NormalizedName() {
					// skip installing self
					continue
				}
				for _, skip := range skipLibs {
					if skip == pkg.Name {
						continue PYPI
					}
				}
				remotePkgs = append(remotePkgs, pkg.PyPiName())
			}
		}
	}
	type depList struct {
		name     string
		packages []string
	}
	dbrDepsName := "remote cluster"
	dependencyLists := []depList{
		{dbrDepsName, remotePkgs},
		{"install_requires", d.InstallRequires},
		{"tests_require", d.TestsRequire},
	}
	for _, deps := range dependencyLists {
		for _, dep := range deps.packages {
			if env.Has(dep) {
				continue
			}
			status(fmt.Sprintf("Installing %s in virtualenv (%s)", dep, deps.name))
			_, err = s.Pip(ctx, "install", "--prefer-binary", dep)
			if err != nil && deps.name == dbrDepsName &&
				strings.Contains(err.Error(), "Could not find a version") {
				continue
			}
			if err != nil {
				return fmt.Errorf("%s: %w", dep, err)
			}
			// repeatedly run pip freeze so that we potentially have less installs
			env, err = s.Freeze(ctx)
			if err != nil {
				return fmt.Errorf("pip freeze: %s", err)
			}
		}
	}
	return nil
}

var errNoCluster = errors.New("no development cluster")

func (s *SetupDotPy) runtimeInfo(ctx context.Context, prj flavor.Project,
	status func(string)) (*dbr.RuntimeInfo, error) {
	clusterId, err := prj.GetDevelopmentClusterId(ctx)
	if err != nil && errors.As(err, &apierr.APIError{}) {
		return nil, err
	}
	if err != nil {
		return nil, errNoCluster
	}
	return dbr.GetRuntimeInfo(ctx, prj.WorkspacesClient(), clusterId, status)
}

func (s *SetupDotPy) Freeze(ctx context.Context) (Environment, error) {
	out, err := s.Pip(ctx, "freeze")
	if err != nil {
		return nil, err
	}
	env := Environment{}
	deps := strings.Split(out, "\n")
	for _, raw := range deps {
		env = append(env, DependencyFromSpec(raw))
	}
	return env, nil
}

func (s *SetupDotPy) LocalArtifacts(ctx context.Context, prj flavor.Project) (flavor.Artifacts, error) {
	dist, err := s.readDistribution(ctx, prj)
	if err != nil {
		return nil, err
	}
	all := flavor.Artifacts{}
	// install dependencies for the wheel to run
	for _, dependency := range dist.InstallRequires {
		if strings.HasPrefix(dependency, "pyspark") {
			// pyspark will conflict with DBR
			continue
		}
		all = append(all, flavor.Artifact{
			Flavor: s,
			Library: libraries.Library{
				Pypi: &libraries.PythonPyPiLibrary{
					Package: dependency,
				},
			},
		})
	}
	s.wheelName = dist.WheelName()
	all = append(all, flavor.Artifact{
		Flavor: s,
		Library: libraries.Library{
			Whl: fmt.Sprintf("%s/.databricks/dist/%s", prj.Root(), s.wheelName),
		},
	})
	return all, nil
}

// Build creates a python wheel, while keeping project root in a clean state, removing the need
// to execute rm -fr dist build *.egg-info after each build
func (s *SetupDotPy) Build(ctx context.Context, prj flavor.Project, status func(string)) error {
	status(fmt.Sprintf("Building %s", s.wheelName))
	ctx = spawn.WithRoot(ctx, filepath.Dir(s.setupPyLoc(prj)))
	_, err := s.Py(ctx, "setup.py",
		// see https://github.com/pypa/setuptools/blob/main/setuptools/_distutils/command/build.py#L23-L31
		"build", "--build-lib=.databricks/build/lib", "--build-base=.databricks/build",
		// see https://github.com/pypa/setuptools/blob/main/setuptools/command/egg_info.py#L167-L168
		"egg_info", "--egg-base=.databricks",
		// see https://github.com/pypa/wheel/blob/main/src/wheel/bdist_wheel.py#L140
		"bdist_wheel", "--dist-dir=.databricks/dist")
	return err
}

// Py calls project-specific Python interpreter from the virtual env from project root dir
func (s *SetupDotPy) Py(ctx context.Context, args ...string) (string, error) {
	if s.venv == "" {
		return "", fmt.Errorf("virtualenv not detected")
	}
	out, err := spawn.ExecAndPassErr(ctx, fmt.Sprintf("%s/bin/python3", s.venv), args...)
	if err != nil {
		// current error message chain is longer:
		// failed to call {pyExec} __non_existing__.py: {pyExec}: can't open
		// ... file '{pwd}/__non_existing__.py': [Errno 2] No such file or directory"
		// probably we'll need to make it shorter:
		// can't open file '$PWD/__non_existing__.py': [Errno 2] No such file or directory
		return "", err
	}
	return strings.Trim(string(out), "\n\r"), nil
}

func (s *SetupDotPy) Pip(ctx context.Context, args ...string) (string, error) {
	return s.Py(ctx, append([]string{"-m", "pip"}, args...)...)
}

// fastDetectVirtualEnv performs very quick detection, by running over top level directories only
func (s *SetupDotPy) fastDetectVirtualEnv(root string) (string, error) {
	wdf, err := os.Open(root)
	if err != nil {
		return "", err
	}
	files, err := wdf.ReadDir(0)
	if err != nil {
		return "", err
	}
	// virtual env is most likely in dot-directory
	slices.SortFunc(files, func(a, b fs.DirEntry) bool {
		return a.Name() < b.Name()
	})
	for _, v := range files {
		if !v.IsDir() {
			continue
		}
		candidate := fmt.Sprintf("%s/%s", root, v.Name())
		_, err = os.Stat(fmt.Sprintf("%s/pyvenv.cfg", candidate))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		return candidate, nil
	}
	return "", nil
}
