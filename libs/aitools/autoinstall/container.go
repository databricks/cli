package autoinstall

import "os"

// containerMarkers are the files container engines create in every container:
// Docker's /.dockerenv and Podman's /run/.containerenv.
// See https://docs.podman.io/en/latest/markdown/podman-run.1.html (/run/.containerenv).
var containerMarkers = []string{"/.dockerenv", "/run/.containerenv"}

// InContainer reports whether the CLI runs in a Docker or Podman container.
// There the CLI cannot tell whether a terminal forwards keystrokes (`docker
// run -t` without `-i` does not), and the home directory often does not
// outlive the container, so a question asked there may be asked on every run.
func InContainer() bool {
	return anyExists(containerMarkers)
}

// anyExists reports whether any of paths exists.
func anyExists(paths []string) bool {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}
