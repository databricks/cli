package apps

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const DEBUG_PORT = "5678"

// defaultLibraries is the list of libraries that will be installed by default.
// We take the list from here: https://docs.databricks.com/aws/en/dev-tools/databricks-apps/#installed-python-libraries
// We install a newer version of databricks-sql-connector because it does not require compiling dependencies on the user's machine.
// We also install debugpy to enable debugging.
var defaultLibraries = []string{
	"databricks-sql-connector==3.7.3",
	"databricks-sdk==0.33.0",
	"mlflow-skinny==2.16.2",
	"gradio==4.44.0",
	"streamlit==1.38.0",
	"shiny==1.1.0",
	"dash==2.18.1",
	"Flask==3.0.3",
	"fastapi==0.115.0",
	"uvicorn[standard]==0.30.6",
	"gunicorn==23.0.0",
	"dash-ag-grid==31.2.0",
	"dash-mantine-components==0.14.4",
	"dash-bootstrap-components==1.6.0",
	"plotly==5.24.1",
	"plotly-resampler==0.10.0",
	"debugpy",
}

type PythonApp struct {
	ctx    context.Context
	config *Config
	spec   *AppSpec
	uvArgs []string
}

func NewPythonApp(ctx context.Context, config *Config, spec *AppSpec) *PythonApp {
	if config.DebugPort == "" {
		config.DebugPort = DEBUG_PORT
	}
	return &PythonApp{ctx: ctx, config: config, spec: spec}
}

// PrepareEnvironment creates a Python virtual environment using uv and installs required dependencies.
// It first creates a virtual environment, then installs default libraries specified in defaultLibraries,
// and finally installs any additional requirements from requirements.txt if it exists.
// Returns an error if any step fails.
func (p *PythonApp) PrepareEnvironment() error {
	// Create venv first
	venvArgs := []string{"uv", "venv"}
	if err := runCommand(p.ctx, p.config.AppPath, venvArgs); err != nil {
		return err
	}

	// Install default libraries
	installArgs := append([]string{"uv", "pip", "install"}, defaultLibraries...)
	if err := runCommand(p.ctx, p.config.AppPath, installArgs); err != nil {
		return err
	}

	// Install requirements if they exist
	if _, err := os.Stat(filepath.Join(p.config.AppPath, "requirements.txt")); err == nil {
		// We also execute command with CWD set at p.config.AppPath
		// so we can just path local path to requirements.txt here
		reqArgs := []string{"uv", "pip", "install", "-r", "requirements.txt"}
		if err := runCommand(p.ctx, p.config.AppPath, reqArgs); err != nil {
			return err
		}
	}

	// Set up run args
	p.uvArgs = []string{"uv", "run"}
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
		files, err := filepath.Glob(filepath.Join(spec.config.AppPath, "*.py"))
		if err != nil {
			return nil, fmt.Errorf("Error reading source code directory: %w", err)
		}

		if len(files) > 0 {
			spec.Command = []string{"python", files[0]}
		}

		if len(spec.Command) == 0 {
			return nil, errors.New("No python file found")
		}

	} else {
		// Replace port bash style with the one in the config
		// We just match the behavior of the Databricks runtime here
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
		spec.Command = append([]string{"python", "-m", "debugpy", "--listen", p.config.DebugPort, "-m"}, spec.Command...)
	} else {
		spec.Command = append([]string{"python", "-m", "debugpy", "--listen", p.config.DebugPort}, spec.Command[1:]...)
	}
}
