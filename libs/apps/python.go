package apps

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const DEBUG_PORT = "5678"

var defaultLibraries = []string{
	"Flask==3.0.3",
	"streamlit==1.38.0",
	"gradio==4.44.0",
	"uvicorn[standard]==0.30.6",
	"databricks-sdk==0.33.0",
	"databricks-sql-connector==3.7.3",
	"debugpy",
}

type PythonApp struct {
	config *Config
	spec   *AppSpec
	venv   bool
}

func NewPythonApp(config *Config, spec *AppSpec) *PythonApp {
	return &PythonApp{config: config, spec: spec}
}

// PrepareEnvironment prepares the environment for running the app. It first checks if the app is in a virtual environment.
// If not, it creates a virtual environment and installs the required libraries. It then installs the libraries from
// requirements.txt if it exists.
func (p *PythonApp) PrepareEnvironment() error {
	// First check that we are not already in virtual environment when we execute CLI command
	// by checking if VIRTUAL_ENV is set
	if os.Getenv("VIRTUAL_ENV") == "" {
		// Check if .venv exists
		_, err := os.Stat(filepath.Join(p.config.AppPath, ".venv"))
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			}

			err := run([]string{"uv", "venv"}, nil)
			if err != nil {
				return err
			}
		}
	}

	env := []string{"VIRTUAL_ENV=" + filepath.Join(p.config.AppPath, ".venv")}

	// Install tools we need to run the app
	args := append([]string{"uv", "pip", "install"}, defaultLibraries...)
	err := run(args, env)
	if err != nil {
		return err
	}

	// Install the requirements from requirements.txt if exists
	_, err = os.Stat(filepath.Join(p.config.AppPath, "requirements.txt"))
	if err == nil {
		err := run([]string{"uv", "pip", "install", "-r", "requirements.txt"}, env)
		if err != nil {
			return err
		}
	}

	p.venv = true
	return nil
}

func run(args, env []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GetCommand returns the command to run the app. If the spec has a command, it is returned.
// If not, the function looks for a python file in the app directory and returns a command
// to run that file. If the app is in a virtual environment, the command is modified to point
// to the python binary in the virtual environment.
func (p *PythonApp) GetCommand(debug bool) ([]string, error) {
	spec := p.spec
	// if no spec, find python file and use it to run app
	if len(spec.Command) == 0 {
		// find python file
		files, err := os.ReadDir(spec.config.AppPath)
		if err != nil {
			return nil, errors.New("Error reading source code directory")
		}

		for _, file := range files {
			// we grab the first python file we find
			if strings.HasSuffix(file.Name(), ".py") {
				spec.Command = []string{"python", file.Name()}
			}
		}

		if len(spec.Command) == 0 {
			return nil, errors.New("No python file found")
		}

	} else {
		// replace port bash style with the one in the config
		for i, cd := range spec.Command {
			if strings.Contains(cd, "$DATABRICKS_APP_PORT") {
				spec.Command[i] = strings.ReplaceAll(cd, "$DATABRICKS_APP_PORT", strconv.Itoa(spec.config.Port))
			}
		}
	}

	if debug {
		p.enableDebugging()
	}

	// if we are in a virtual environment, we need to change the command to point to the python binary in the virtual environment
	if os.Getenv("VIRTUAL_ENV") != "" || p.venv {
		// On windows, the python binary is in Scripts directory
		if runtime.GOOS == "windows" {
			spec.Command[0] = filepath.Join(p.venvPath(), "Scripts", spec.Command[0])
		} else {
			spec.Command[0] = filepath.Join(p.venvPath(), "bin", spec.Command[0])
		}
	}

	return spec.Command, nil
}

// enableDebugging enables debugging for the app by starting the app with debugpy
// listening on the specified port. it modifies the spec.Command to include the
// debugpy command.
func (p *PythonApp) enableDebugging() {
	spec := p.spec
	if spec.Command[0] != "python" {
		spec.Command = append([]string{"python", "-m", "debugpy", "--listen", DEBUG_PORT, "-m"}, spec.Command...)
	} else {
		spec.Command = append([]string{"python", "-m", "debugpy", "--listen", DEBUG_PORT}, spec.Command[1:]...)
	}
}

func (p *PythonApp) venvPath() string {
	return filepath.Join(p.config.AppPath, ".venv")
}
