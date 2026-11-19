package sandbox_test

import (
	"os/exec"
	"runtime"
	"testing"

	"github.com/databricks/cli/experimental/ssh/sandbox"
	"github.com/stretchr/testify/require"
)

func TestServerLauncherReapsSessionChildren(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("sandbox child cleanup uses Linux subreaping")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("requires Python 3")
	}
	const harness = `import os, signal, subprocess, sys, time
from pathlib import Path
launcher, directory, mode = sys.argv[1:]
pidfile = Path(directory) / "child.pid"
child = "import signal,time; signal.signal(signal.SIGTERM, signal.SIG_IGN); time.sleep(300)"
server = '''import subprocess, sys, time
from pathlib import Path
child = subprocess.Popen([sys.executable, "-c", sys.argv[1]], start_new_session=True,
                         stdin=subprocess.DEVNULL, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
Path(sys.argv[2]).write_text(str(child.pid))
print("READY", flush=True)
if sys.argv[3] == "exit":
    sys.exit(23)
time.sleep(300)
'''
neighbor = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(300)"])
process = subprocess.Popen([sys.executable, "-c", launcher, directory,
                           sys.executable, "-c", server, child, str(pidfile), mode],
                          stdout=subprocess.PIPE, stderr=subprocess.PIPE, universal_newlines=True)
try:
    assert process.stdout.readline() == "READY\n"
    pid = int(pidfile.read_text())
    if mode == "terminate":
        process.terminate()
    stdout, stderr = process.communicate(timeout=10)
    assert process.returncode == (143 if mode == "terminate" else 23), stderr
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        pass
    else:
        raise AssertionError("SSH session child survived launcher cleanup")
    assert neighbor.poll() is None, "cleanup stopped another invocation's process"
finally:
    if process.poll() is None:
        process.kill()
        process.wait()
    if pidfile.exists():
        try:
            os.kill(int(pidfile.read_text()), signal.SIGKILL)
        except ProcessLookupError:
            pass
    neighbor.kill()
    neighbor.wait()
`
	for _, mode := range []string{"exit", "terminate"} {
		t.Run(mode, func(t *testing.T) {
			command := exec.CommandContext(t.Context(), "python3", "-c", harness, sandbox.ServerLauncherScriptForTest, t.TempDir(), mode)
			output, err := command.CombinedOutput()
			require.NoError(t, err, string(output))
		})
	}
}
