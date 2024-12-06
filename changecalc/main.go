/*
When passed list of files as arguments, figures out the all the packages that need to be tested
(by following dependencies) and outputs them one per line. The output is suitable to be passed
to "go test" as parameters.

If no arguments were passed, runs "git diff main --name-only -- ." and reads the list of there.
*/
package main

import (
	"bytes"
	"fmt"
	"iter"
	"log"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/go/packages"
	"gopkg.in/yaml.v2"
)

type Config struct {
	BaseBranch string   `yaml:"base_branch"`
	GoModName  string   `yaml:"go_mod"`
	ResetList  []string `yaml:"reset_list"`
}

type ParsedConfig struct {
	Config   Config
	ResetMap map[string]struct{}
}

var empty = struct{}{}

const CONFIG_NAME = "changecalc.yml"

func main() {
	var err error
	initialPaths := os.Args[1:]
	log.SetFlags(0)

	config := readConfig()
	//fmt.Fprintf(os.Stderr, "config=%v\n", config)

	if len(initialPaths) == 0 {
		initialPaths, err = GetChangedFiles(config.Config.BaseBranch)
		if err != nil {
			log.Fatal(err)
		}
	}

	//fmt.Fprintf(os.Stderr, "initialPaths=%v\n", initialPaths)

	moduleName, err := readModuleName(config.Config.GoModName)
	if err != nil {
		log.Fatalf("Failed to read module name from %s: %v", config.Config.GoModName, err)
	}

	goPackages := make(map[string]struct{}, 128)
	testDirs := make(map[string]struct{}, 128)
	testdata := fmt.Sprintf("%ctestdata%c", filepath.Separator, filepath.Separator)

	for _, p := range initialPaths {
		p = filepath.Clean(p)
		if _, ok := config.ResetMap[p]; ok {
			log.Fatalf("Found %s in changed paths, will not proceed.\n", p)
		}

		dir := filepath.Dir(p)
		if strings.HasSuffix(p, "_test.go") {
			testDirs[dir] = empty
		} else if strings.HasSuffix(p, ".go") {
			pkg := filepath.Join(moduleName, dir)
			goPackages[pkg] = empty
		} else if strings.Contains(p, testdata) {
			items := strings.Split(p, testdata)
			testDirs[items[0]] = empty
		} else {
			// we're not parsing go:embed, instead assuming that data file is read by the closest go package
			realDir := findNearestGoPackage(dir)
			if realDir != "" {
				pkg := filepath.Join(moduleName, realDir)
				goPackages[pkg] = empty
			}
		}
	}

	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedImports | packages.NeedDeps,
		Dir:  ".",
	}

	allPkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		log.Fatalf("Failed to load packages: %v", err)
	}

	reverseDeps := make(map[string][]string)
	for _, pkg := range allPkgs {
		for imported := range pkg.Imports {
			reverseDeps[imported] = append(reverseDeps[imported], pkg.PkgPath)
		}
	}

	dependentPackages := findDependents(maps.Keys(goPackages), reverseDeps)

	for pkg := range dependentPackages {
		var dir string
		if pkg == moduleName {
			dir = "."
		} else if strings.HasPrefix(pkg, moduleName+"/") {
			dir = pkg[len(moduleName)+1:]
		} else {
			fmt.Fprintf(os.Stderr, "Internal error: %s\n", pkg)
			continue
		}
		testDirs[dir] = empty
	}

	belongsCache := make(map[string]bool, len(testDirs)*4)

	for dir := range testDirs {
		if !checkIfBelongs(dir, belongsCache, config.Config.GoModName) {
			//fmt.Fprintf(os.Stderr, "Excluding %s: part of another go module\n", dir)
			continue
		}
		if dir != "." {
			dir = "./" + dir
		}
		fmt.Println(dir)
	}
}

func readConfig() ParsedConfig {
	config := Config{}

	data, err := os.ReadFile(CONFIG_NAME)
	if err != nil {
		log.Fatalf("reading config file: %s: %v", CONFIG_NAME, err)
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatal(err)
	}

	resetMap := make(map[string]struct{}, len(config.ResetList))
	for _, item := range config.ResetList {
		resetMap[item] = empty
	}

	return ParsedConfig{Config: config, ResetMap: resetMap}
}

// Returns true if this directory belongs to the current go module (identified by MODULE_NAME_SOURCE file)
func checkIfBelongs(dir string, cache map[string]bool, rootIdName string) bool {
	if dir == "." || dir == "" {
		return true
	}

	val, ok := cache[dir]
	if ok {
		return val
	}

	result := false

	//fmt.Fprintf(os.Stderr, "Checking %s for go.mod\n", dir)
	_, err := os.Stat(filepath.Join(dir, rootIdName))
	if err != nil {
		// assuming it's not-found
		result = checkIfBelongs(filepath.Dir(dir), cache, rootIdName)
	}
	// If we found go.mod, it's a root of another module.
	// "go test" will fail if we pass this directory:
	// % go test ./bundle/internal/tf/codegen/schema
	// main module (github.com/databricks/cli) does not contain package github.com/databricks/cli/bundle/internal/tf/codegen/schema
	cache[dir] = result
	return result
}

// readModuleName parses the go.mod file to extract the module name.
func readModuleName(goModPath string) (string, error) {
	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", err
	}
	modFile, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return "", err
	}
	return modFile.Module.Mod.Path, nil
}

// resolvePackages converts file paths to fully qualified package names.
// Recursively find all packages that depend on the given list
func findDependents(initialPackages iter.Seq[string], reverseDeps map[string][]string) map[string]struct{} {
	visited := make(map[string]struct{})
	var visit func(pkg string)
	visit = func(pkg string) {
		if _, seen := visited[pkg]; seen {
			return
		}
		visited[pkg] = struct{}{}
		for _, dependent := range reverseDeps[pkg] {
			visit(dependent)
		}
	}
	for pkg := range initialPackages {
		visit(pkg)
	}
	return visited
}

// findNearestGoPackage returns the closest parent (or itself) directory
// that contains at least one *.go file. If no such directory is found,
// it returns an empty string.
func findNearestGoPackage(directory string) string {
	for {
		if hasGoFiles(directory) {
			return directory
		}

		parent := filepath.Dir(directory)

		if parent == directory {
			break
		}

		directory = parent
	}

	return ""
}

// hasGoFiles checks if a directory contains any .go files
func hasGoFiles(directory string) bool {
	files, err := os.ReadDir(directory)
	if err != nil {
		return false
	}

	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
			return true
		}
	}
	return false
}

// GetChangedFiles compares the current branch to the base branch
// and returns a slice of file paths that have been modified.
func GetChangedFiles(baseBranch string) ([]string, error) {
	command := []string{"git", "diff", baseBranch, "--name-only", "--", "."}
	cmd := exec.Command(command[0], command[1:]...)

	// Capture the command's output
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to execute %s: %w", command, err)
	}

	trimmedOut := strings.TrimSpace(out.String())
	if trimmedOut == "" {
		return []string{}, nil
	}

	changedFiles := strings.Split(trimmedOut, "\n")
	return changedFiles, nil
}
