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

	// If path starts with /, it's a remote absolute path
	return !path.IsAbs(p)
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
