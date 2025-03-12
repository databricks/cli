package apps

import (
	"errors"
	"os"
	"path/filepath"
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
	uvArgs []string
}

func NewPythonApp(config *Config, spec *AppSpec) *PythonApp {
	return &PythonApp{config: config, spec: spec}
}

// PrepareEnvironment prepares the environment for running the app. It first checks if the app is in a virtual environment.
// If not, it creates a virtual environment and installs the required libraries. It then installs the libraries from
// requirements.txt if it exists.
func (p *PythonApp) PrepareEnvironment() error {
	args := append([]string{"uv", "run", "--with"}, strings.Join(defaultLibraries, ","))

	// Install the requirements from requirements.txt if exists
	_, err := os.Stat(filepath.Join(p.config.AppPath, "requirements.txt"))
	if err == nil {
		args = append(args, "--with-requirements", filepath.Join(p.config.AppPath, "requirements.txt"))
	}

	p.uvArgs = args
	return nil
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

	if p.uvArgs != nil {
		spec.Command = append(p.uvArgs, spec.Command...)
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
