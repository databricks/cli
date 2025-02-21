package main

import (
	"os"
	"path/filepath"

	"github.com/databricks/cli/libs/daemon"
)

func main() {
	tmpDir := os.Args[1]

	d := daemon.Daemon{
		PidFilePath: filepath.Join(tmpDir, "child.pid"),
		Executable:  "python3",
		// The server script writes the port number the server is listening on
		// to the specified file.
		Args: []string{"./internal/parent_process/server.py", filepath.Join(tmpDir, "port.txt")},
	}

	err := d.Start()
	if err != nil {
		panic(err)
	}

	err = d.Release()
	if err != nil {
		panic(err)
	}
}
