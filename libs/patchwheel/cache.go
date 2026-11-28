package patchwheel

import "path/filepath"

// CacheDir returns the directory under a bundle's local state directory that holds
// the patched copies of one artifact's wheels of one distribution. "bundle plan" and
// "bundle deploy" write there, and "bundle deploy --plan" reads the same files back.
func CacheDir(stateDir, artifactName, distribution string) string {
	return filepath.Join(stateDir, "patched_wheels", artifactName+"_"+distribution)
}
