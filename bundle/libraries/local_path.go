package libraries

import (
	"net/url"
	"path"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/compute"
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

	// If the path has another scheme, it's a remote path.
	if isRemoteStorageScheme(p) {
		return false
	}

	// if path starts with . or .., it's a local path
	if strings.HasPrefix(p, ".") || strings.HasPrefix(p, "..") {
		return true
	}

	// If the path is absolute, it's might be a remote path.
	if path.IsAbs(p) {
		possiblePrefixes := []string{
			"/Workspace", "/Users", "/mnt", "/dbfs",
			"/Volumes", "/Shared",
		}

		for _, prefix := range possiblePrefixes {
			if strings.HasPrefix(p, prefix) {
				return false
			}
		}
	}

	return true
}

// IsLibraryLocal returns true if the specified library or environment dependency
// should be interpreted as a local path.
// We use this to check if the dependency in environment spec is local or that library is local.
// We can't use IsLocalPath beacuse environment dependencies can be
// a pypi package name which can be misinterpreted as a local path by IsLocalPath.
func IsLibraryLocal(dep string) bool {
	possiblePrefixes := []string{
		".", "..",
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
	if path.Ext(dep) == "" {
		return false
	}

	return IsLocalPath(dep)
}

func isRemoteStorageScheme(path string) bool {
	url, err := url.Parse(path)
	if err != nil {
		return false
	}

	if url.Scheme == "" {
		return false
	}

	// If the path starts with scheme:/ format, it's a correct remote storage scheme
	return strings.HasPrefix(path, url.Scheme+":/")
}

// IsLocalLibrary returns true if the specified library refers to a local path.
func IsLocalLibrary(library *compute.Library) bool {
	path := libraryPath(library)
	if path == "" {
		return false
	}

	return IsLocalPath(path)
}
