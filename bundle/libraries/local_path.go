package libraries

import (
	"net/url"
	"path"
	"regexp"
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

	// If the path has another scheme, it's a remote path.
	if isRemoteStorageScheme(p) {
		return false
	}

	// If path starts with /, it's a remote absolute path
	return !path.IsAbs(p)
}

// IsLibraryLocal returns true if the specified library or environment dependency
// should be interpreted as a local path.
// We use this to check if the dependency in environment spec is local or that library is local.
// We can't use IsLocalPath beacuse environment dependencies can be
// a pypi package name which can be misinterpreted as a local path by IsLocalPath.
func IsLibraryLocal(dep string) bool {
	if dep == "" {
		return false
	}

	possiblePrefixes := []string{
		".",
	}

	for _, prefix := range possiblePrefixes {
		if strings.HasPrefix(dep, prefix) {
			return true
		}
	}

	// If the dependency starts with --, it's a pip flag option which is a valid
	// entry for environment dependencies but not a local path
	if containsPipFlag(dep) {
		return false
	}

	// If the dependency is a requirements file, it's not a valid local path
	if strings.HasPrefix(dep, "-r") {
		return false
	}

	// If the dependency has no extension, it's a PyPi package name
	if isPackage(dep) {
		return false
	}

	return IsLocalPath(dep)
}

func containsPipFlag(input string) bool {
	re := regexp.MustCompile(`--[a-zA-Z0-9-]+`)
	return re.MatchString(input)
}

// ^[a-zA-Z0-9\-_]+: Matches the package name, allowing alphanumeric characters, dashes (-), and underscores (_).
// \[.*\])?: Optionally matches any extras specified in square brackets, e.g., [security].
// ((==|!=|<=|>=|~=|>|<)\d+(\.\d+){0,2}(\.\*)?): Optionally matches version specifiers, supporting various operators (==, !=, etc.) followed by a version number (e.g., 2.25.1).
// ,?: Optionally matches a comma (,) at the end of the specifier which is used to separate multiple specifiers.
// There can be multiple version specifiers separated by commas or no specifiers.
// Spec for package name and version specifier: https://pip.pypa.io/en/stable/reference/requirement-specifiers/
var packageRegex = regexp.MustCompile(`^[a-zA-Z0-9\-_]+\s?(\[.*\])?\s?((==|!=|<=|>=|~=|==|>|<)\s?\d+(\.\d+){0,2}(\.\*)?,?)*$`)

func isPackage(name string) bool {
	if packageRegex.MatchString(name) {
		return true
	}

	return isUrlBasedLookup(name)
}

func isUrlBasedLookup(name string) bool {
	parts := strings.Split(name, " @ ")
	if len(parts) != 2 {
		return false
	}

	return packageRegex.MatchString(parts[0]) && isRemoteStorageScheme(parts[1])
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
