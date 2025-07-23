package internal

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/google/uuid"
)

type CmdRunner interface {
	SetDir(dir string)
	SetArgs(args []string)
	SetEnv(env []string)
	AddEnv(env string)
	Start() error
	Run() error
	Wait() error
	GetExitCode() int
	Output() io.ReadCloser
}

type DbrRunner struct {
	dir  string
	args []string
	env  []string

	w *databricks.WorkspaceClient

	// Internal state for cleanup
	tempWorkspaceDir string
	runId            int64

	// Output storage
	outputContent io.ReadCloser
	exitCode      int
}

func NewDbrRunner(w *databricks.WorkspaceClient) *DbrRunner {
	return &DbrRunner{
		w: w,
	}
}

func (r *DbrRunner) SetDir(dir string) {
	r.dir = dir
}

func (r *DbrRunner) SetArgs(args []string) {
	r.args = args
}

func (r *DbrRunner) AddEnv(env string) {
	if r.env == nil {
		r.env = []string{}
	}
	r.env = append(r.env, env)
}

func (r *DbrRunner) SetEnv(env []string) {
	r.env = env
}

func (r *DbrRunner) Start() error {
	ctx := context.Background()

	// Step 1: Create temporary workspace directory
	if err := r.createTempWorkspaceDir(ctx); err != nil {
		return fmt.Errorf("failed to create temporary workspace directory: %w", err)
	}

	// Step 2: Upload files from local directory to workspace
	if err := r.uploadFiles(ctx); err != nil {
		return fmt.Errorf("failed to upload files: %w", err)
	}

	// Step 3: Create and upload Python runner script
	if err := r.uploadRunnerScript(ctx); err != nil {
		return fmt.Errorf("failed to upload runner script: %w", err)
	}

	// Step 4: Submit and run the job
	if err := r.submitJob(ctx); err != nil {
		return fmt.Errorf("failed to submit job: %w", err)
	}

	return nil
}

func (r *DbrRunner) Wait() error {
	ctx := context.Background()

	// Ensure cleanup happens
	defer r.Cleanup(ctx)

	// Fetch output files and store in memory
	if err := r.fetchOutputFiles(ctx); err != nil {
		return fmt.Errorf("failed to fetch output files: %w", err)
	}

	return nil
}

func (r *DbrRunner) Run() error {
	err := r.Start()
	if err != nil {
		return err
	}
	return r.Wait()
}

func (r *DbrRunner) Output() io.ReadCloser {
	return r.outputContent
}

func (r *DbrRunner) GetExitCode() int {
	return r.exitCode
}

// SetCancelFunc is a no-op for DbrRunner since Databricks jobs handle cancellation differently
func (r *DbrRunner) SetCancelFunc(cancelFunc func() error) {
	// No-op: Databricks jobs use their own timeout/cancellation mechanism
}

// Kill attempts to cancel the running Databricks job
func (r *DbrRunner) Kill() error {
	if r.runId == 0 {
		return nil // No job to cancel
	}

	ctx := context.Background()
	_, err := r.w.Jobs.CancelRun(ctx, jobs.CancelRun{RunId: r.runId})
	return err
}

// GetProcess returns nil since there's no local process for Databricks jobs
func (r *DbrRunner) GetProcess() *os.Process {
	return nil
}

// GetWriter returns nil since DbrRunner doesn't support streaming output
func (r *DbrRunner) GetWriter() io.WriteCloser {
	return nil
}

func (r *DbrRunner) createTempWorkspaceDir(ctx context.Context) error {
	// Get current user to create directory under their home
	me, err := r.w.CurrentUser.Me(ctx)
	if err != nil {
		return err
	}

	// Create temporary directory name
	tempDirName := fmt.Sprintf("dbr-runner-temp-%s", uuid.New().String())
	r.tempWorkspaceDir = fmt.Sprintf("/Workspace/Users/%s/%s", me.UserName, tempDirName)

	// Create the directory
	err = r.w.Workspace.MkdirsByPath(ctx, r.tempWorkspaceDir)
	if err != nil {
		return err
	}

	return nil
}

func (r *DbrRunner) uploadFiles(ctx context.Context) error {
	// Create filer for workspace directory
	f, err := filer.NewWorkspaceFilesClient(r.w, r.tempWorkspaceDir)
	if err != nil {
		return err
	}

	// Walk through the local directory and upload all files
	return filepath.Walk(r.dir, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil // Skip directories, filer will create them as needed
		}

		// Calculate relative path from source directory
		relPath, err := filepath.Rel(r.dir, localPath)
		if err != nil {
			return err
		}

		// Convert to slash-separated path for workspace
		workspacePath := filepath.ToSlash(relPath)

		// Open local file
		localFile, err := os.Open(localPath)
		if err != nil {
			return err
		}
		defer localFile.Close()

		// Upload to workspace
		return f.Write(ctx, workspacePath, localFile, filer.CreateParentDirectories, filer.OverwriteIfExists)
	})
}

func (r *DbrRunner) uploadRunnerScript(ctx context.Context) error {
	// Create the Python runner script content equivalent to dbr_runner.py
	runnerScript, err := r.generateRunnerScript()
	if err != nil {
		return err
	}

	// Create filer for workspace directory
	f, err := filer.NewWorkspaceFilesClient(r.w, r.tempWorkspaceDir)
	if err != nil {
		return err
	}

	// Upload the runner script
	return f.Write(ctx, "dbr_runner.py", strings.NewReader(runnerScript), filer.OverwriteIfExists)
}

func (r *DbrRunner) generateRunnerScript() (string, error) {
	if len(r.args) == 0 {
		return "", fmt.Errorf("no arguments provided")
	}

	// Generate equivalent Python script to dbr_runner.py but with embedded args
	argsStr := ""
	for i, arg := range r.args {
		if i > 0 {
			argsStr += ", "
		}
		argsStr += fmt.Sprintf("'%s'", arg)
	}

	return fmt.Sprintf(`#!/usr/bin/env python3

import sys
import os
import subprocess

def run_command(args, env_vars):
    """Run the provided command and capture output to files."""
    exit_code = 0
    try:
        # Parse environment variables and add them to the environment
        env = os.environ.copy()
        for env_var in env_vars:
            if '=' in env_var:
                key, value = env_var.split('=', 1)
                env[key] = value

        with open("_internal_output", "w") as output_file:
            result = subprocess.run(
                args,
                stdout=output_file,
                stderr=subprocess.STDOUT,
                text=True,
                cwd=os.getcwd(),
                env=env
            )

        # Capture the exit code
        exit_code = result.returncode

    except Exception as e:
        with open("_internal_output", "w") as output_file:
            output_file.write(f"Error executing command: {e}")
        exit_code = 1

    # Write the exit code to file
    with open("_internal_exit_code", "w") as exit_code_file:
        exit_code_file.write(str(exit_code))

    return

if __name__ == "__main__":
    # Embedded command arguments
    command_args = [%s]
    env_vars = []

    for env in sys.argv[1:]:
        env_vars.append(env)

    run_command(command_args, env_vars)
`, argsStr), nil
}

func (r *DbrRunner) submitJob(ctx context.Context) error {
	// Combine command arguments and environment variables for Python script
	var pythonArgs []string
	pythonArgs = append(pythonArgs, r.env...)

	// Create job submission request for serverless compute
	submitReq := jobs.SubmitRun{
		RunName: "dbr-runner-job",
		Tasks: []jobs.SubmitTask{
			{
				TaskKey: "run_command",
				SparkPythonTask: &jobs.SparkPythonTask{
					PythonFile: path.Join(r.tempWorkspaceDir, "dbr_runner.py"),
					Parameters: pythonArgs,
				},
				EnvironmentKey: "dbr-runner-env",
			},
		},
		Environments: []jobs.JobEnvironment{
			{
				EnvironmentKey: "dbr-runner-env",
				Spec: &compute.Environment{
					EnvironmentVersion: "2", // Use environment version 2 for Python 3.x
				},
			},
		},
		TimeoutSeconds: 1800, // 30 minutes timeout
	}

	// Submit the job run
	waiter, err := r.w.Jobs.Submit(ctx, submitReq)
	if err != nil {
		return err
	}

	r.runId = waiter.RunId

	// Wait for completion
	// TODO: Use consistent timeout.
	run, err := waiter.GetWithTimeout(time.Hour)
	if err != nil {
		return err
	}

	// Check if the run was successful
	if run.State.LifeCycleState == jobs.RunLifeCycleStateInternalError ||
		(run.State.LifeCycleState == jobs.RunLifeCycleStateTerminated &&
			run.State.ResultState == jobs.RunResultStateFailed) {
		return fmt.Errorf("job run failed: %s", run.State.StateMessage)
	}

	return nil
}

func (r *DbrRunner) fetchOutputFiles(ctx context.Context) error {
	// Create filer to read output files from workspace
	f, err := filer.NewWorkspaceFilesClient(r.w, r.tempWorkspaceDir)
	if err != nil {
		return err
	}

	// Fetch combined output content
	outputReader, err := f.Read(ctx, "_internal_output")
	if err != nil {
		r.outputContent = io.NopCloser(strings.NewReader(""))
	} else {
		r.outputContent = outputReader
		outputReader.Close()
		if err != nil {
			return err
		}
	}

	// Fetch exit code
	exitCodeReader, err := f.Read(ctx, "_internal_exit_code")
	if err != nil {
		// If file doesn't exist, set default exit code to 0
		r.exitCode = 0
	} else {
		exitCodeBytes, err := io.ReadAll(exitCodeReader)
		exitCodeReader.Close()
		if err != nil {
			return err
		}

		// Parse exit code as integer
		r.exitCode, err = strconv.Atoi(strings.TrimSpace(string(exitCodeBytes)))
		if err != nil {
			// If parsing fails, default to 0
			r.exitCode = 0
		}
	}

	return nil
}

// Cleanup removes the temporary workspace directory
func (r *DbrRunner) Cleanup(ctx context.Context) error {
	if r.tempWorkspaceDir != "" {
		err := r.w.Workspace.Delete(ctx, workspace.Delete{
			Path:      r.tempWorkspaceDir,
			Recursive: true,
		})
		// Don't return error if directory already doesn't exist
		if err != nil {
			return fmt.Errorf("failed to cleanup temporary directory %s: %w", r.tempWorkspaceDir, err)
		}
	}
	return nil
}
