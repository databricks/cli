package libraries

import (
	"fmt"
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
	// If the path has the explicit file scheme, check if it's an absolute path.
	// file:///absolute/path is a runtime container path (remote).
	// file://relative/path is a local path to be uploaded.
	if strings.HasPrefix(p, "file://") {
		// Extract the path after file://
		pathAfterScheme := strings.TrimPrefix(p, "file://")
		// If it's an absolute path (starts with /), it's a runtime path, not local
		return !path.IsAbs(pathAfterScheme)
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

	// If the dependency starts with - or --, it's a pip flag option which is a valid
	// entry for environment dependencies. Even though the path in the flag can be local we don't return true in this function anyway
	// and don't treat such path as a local library path.
	// Instead we handle translation of these paths in a separate code path in TranslatePath mutator.
	if containsPipFlag(dep) {
		return false
	}

	// If the dependency has no extension, it's a PyPi package name
	if isPackage(dep) {
		return false
	}

	return IsLocalPath(dep)
}

var PipFlagsWithLocalPaths = []string{
	"-r",
	"-e",
}

func IsLocalPathInPipFlag(dep string) (string, string, bool) {
	for _, flag := range PipFlagsWithLocalPaths {
		depWithoutFlag, ok := strings.CutPrefix(dep, flag+" ")
		if ok {
			depWithoutFlag = strings.TrimSpace(depWithoutFlag)
			return depWithoutFlag, flag, IsLocalPath(depWithoutFlag)
		}
	}

	return "", "", false
}

func containsPipFlag(input string) bool {
	// Trailing space means the the flag takes an argument or there's multiple arguments in input
	// Alternatively it could be a flag with no argument and no space after it
	// For example: -r myfile.txt or --index-url http://myindexurl.com or -i
	re := regexp.MustCompile(`(^|\s+)--?[a-zA-Z0-9-]+(([\s|=]+)|$)`)
	return re.MatchString(input)
}

// pep440Regex is a regular expression that matches PEP 440 version specifiers.
// https://peps.python.org/pep-0440/#public-version-identifiers
const pep440Regex = `(?P<epoch>\d+!)?(?P<release>\d+(\.\d+)*)(?P<pre>[-_.]?(a|b|rc)\d+)?(?P<post>[-_.]?post\d+)?(?P<dev>[-_.]?dev\d+)?(?P<local>\+[a-zA-Z0-9]+([-_.][a-zA-Z0-9]+)*)?`

// ^[a-zA-Z0-9\-_]+: Matches the package name, allowing alphanumeric characters, dashes (-), and underscores (_).
// \[.*\])?: Optionally matches any extras specified in square brackets, e.g., [security].
// ((==|!=|<=|>=|~=|>|<)\s?pep440Regex): Optionally matches version specifiers, supporting various operators (==, !=, etc.) followed by a version number as per PEP440 spec.
// ,?: Optionally matches a comma (,) at the end of the specifier which is used to separate multiple specifiers.
// There can be multiple version specifiers separated by commas or no specifiers.
// Spec for package name and version specifier: https://pip.pypa.io/en/stable/reference/requirement-specifiers/
var packageRegex = regexp.MustCompile(fmt.Sprintf(`^[a-zA-Z0-9\-_]+\s?(\[.*\])?\s?((==|!=|<=|>=|~=|==|>|<)\s?%s,?)*$`, pep440Regex))

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
