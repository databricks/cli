package libraries

import (
	"net/url"
	"path"
	"strings"
)

// IsLocalPath returns true if the specified path indicates that
// it should be interpreted as a path on the local file system.
//
// The following paths are considered local:
//
// - myfile.txt
// - ./myfile.txt
// - ../myfile.txt
// - file:///foo/bar/myfile.txt
//
// The following paths are considered remote:
//
// - dbfs:/mnt/myfile.txt
// - s3:/mybucket/myfile.txt
// - /Users/jane@doe.com/myfile.txt
func IsLocalPath(p string) bool {
	// If the path has the explicit file scheme, it's a local path.
	if strings.HasPrefix(p, "file://") {
		return true
	}

	if IsRemotePath(p) {
		return false
	}

	// If the path is absolute, it's a remote path.
	return !path.IsAbs(p)
}

func IsRemotePath(p string) bool {
	if isRemoteStorageScheme(p) {
		return true
	}

	// if the path is not absolute, it's not a remote path
	if !path.IsAbs(p) {
		return false
	}

	// If the path is absolute, it's might be a remote path.
	possiblePrefixes := []string{
		"/Workspace",
		"/Users",
		"/Volumes",
		"/Shared",
		"/mnt",
		"/dbfs",
	}

	for _, prefix := range possiblePrefixes {
		if strings.HasPrefix(p, prefix) {
			return true
		}
	}

	return false
}

// IsLibraryLocal returns true if the specified library or environment dependency
// should be interpreted as a local path.
// We use this to check if the dependency in environment spec is local or that library is local.
// We can't use IsLocalPath beacuse environment dependencies can be
// a pypi package name which can be misinterpreted as a local path by IsLocalPath.
func IsLibraryLocal(dep string) bool {
	possiblePrefixes := []string{
		".",
	}

	for _, prefix := range possiblePrefixes {
		if strings.HasPrefix(dep, prefix) {
			return true
		}
	}

	// If the dependency is a requirements file, it's not a valid local path
	if strings.HasPrefix(dep, "-r") {
		return false
	}

	// If the dependency has no extension, it's a PyPi package name
	if isPackage(dep) {
		return false
	}

	if IsRemotePath(dep) {
		return false
	}

	return true
}

func isPackage(name string) bool {
	// If the dependency has no extension, it's a PyPi package name
	return path.Ext(name) == ""
}

func isRemoteStorageScheme(path string) bool {
	url, err := url.Parse(path)
	if err != nil {
		return false
	}

	if url.Scheme == "" {
		return false
	}

	// If the path starts with scheme:/ format (not file), it's a correct remote storage scheme
	return strings.HasPrefix(path, url.Scheme+":/") && url.Scheme != "file"
}
