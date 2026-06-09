// Package cligen regenerates the CLI command stubs (cmd/workspace/** and
// cmd/account/**) from the cli.json spec produced by genkit.
//
// It is the CLI-owned replacement for running genkit's cli_v0 generator against
// a universe checkout: it renders the same templates (templates/*.tmpl, copied
// verbatim) against a model decoded from cli.json's "commands" block. It has no
// dependency on genkit or any upstream spec at run time.
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func main() {
	input := flag.String("input", ".codegen/cli.json", "path to the cli.json spec")
	output := flag.String("output", ".", "target directory (CLI repo root)")
	flag.Parse()

	files, err := Generate(*input, *output)
	if err != nil {
		log.Fatalf("cligen: %v", err)
	}
	if err := format(*output, files); err != nil {
		log.Fatalf("cligen: format: %v", err)
	}
	if err := writeGitAttributes(*output, files); err != nil {
		log.Fatalf("cligen: .gitattributes: %v", err)
	}
	for _, f := range files {
		fmt.Println(f)
	}
}

// writeGitAttributes writes .gitattributes marking the generated command files
// as linguist-generated, mirroring genkit: the header comment comes from the
// hand-maintained .gitattributes.manual, followed by the sorted file list.
func writeGitAttributes(dir string, files []string) error {
	header, err := os.ReadFile(filepath.Join(dir, ".gitattributes.manual"))
	if err != nil {
		return err
	}
	sorted := append([]string(nil), files...)
	slices.Sort(sorted)
	var sb strings.Builder
	sb.Write(header)
	for _, f := range sorted {
		sb.WriteString(f)
		sb.WriteString(" linguist-generated=true\n")
	}
	return os.WriteFile(filepath.Join(dir, ".gitattributes"), []byte(sb.String()), 0o644)
}

// format applies the same formatter genkit's cli_v0 generator used to produce
// the committed command files: goimports (prunes the conditionally-unused
// imports the template always emits, e.g. "time") followed by gofmt.
func format(dir string, files []string) error {
	abs := make([]string, len(files))
	for i, f := range files {
		abs[i] = filepath.Join(dir, f)
	}
	// Run goimports from the pinned tools module (tools/go.mod) instead of
	// @latest, so generation is deterministic and works offline.
	modfile := filepath.Join(dir, "tools", "go.mod")
	goimports := append([]string{"tool", "-modfile=" + modfile, "goimports", "-w"}, abs...)
	if out, err := exec.Command("go", goimports...).CombinedOutput(); err != nil {
		return fmt.Errorf("goimports: %v\n%s", err, out)
	}
	if out, err := exec.Command("gofmt", append([]string{"-w"}, abs...)...).CombinedOutput(); err != nil {
		return fmt.Errorf("gofmt: %v\n%s", err, out)
	}
	return nil
}

// cliJSON is the subset of the cli.json contract that command generation needs.
type cliJSON struct {
	Commands *CommandsBlock `json:"commands"`
}

// Generate reads the cli.json spec at jsonPath and writes the command stubs
// under targetDir. It returns the list of files written (relative to targetDir).
func Generate(jsonPath, targetDir string) ([]string, error) {
	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		return nil, err
	}
	var doc cliJSON
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", jsonPath, err)
	}
	if doc.Commands == nil {
		return nil, fmt.Errorf("%s: missing \"commands\" block", jsonPath)
	}
	batch := doc.Commands
	batch.Resolve()

	var filenames []string

	// Batch-level files (mirrors cliv0.Apply order).
	batchFiles := []struct {
		tmplName string
		tmplFile string
		out      string
	}{
		{"cmds-workspace.go.tmpl", "templates/cmds-workspace.go.tmpl", "cmd/workspace/cmd.go"},
		{"groups-workspace.go.tmpl", "templates/groups-workspace.go.tmpl", "cmd/workspace/groups.go"},
		{"cmds-account.go.tmpl", "templates/cmds-account.go.tmpl", "cmd/account/cmd.go"},
		{"groups-account.go.tmpl", "templates/groups-account.go.tmpl", "cmd/account/groups.go"},
	}
	for _, bf := range batchFiles {
		t := parseTemplate(bf.tmplName, bf.tmplFile)
		if err := renderToFile(batch, t, bf.tmplName, filepath.Join(targetDir, bf.out)); err != nil {
			return nil, fmt.Errorf("render %s: %w", bf.out, err)
		}
		filenames = append(filenames, bf.out)
	}

	// Per-service files.
	serviceTmpl := parseTemplate("service", "templates/service.go.tmpl")
	for _, svc := range batch.Services {
		out := serviceFilename(svc)
		err := renderToFile(svc, serviceTmpl, "service.go.tmpl", filepath.Join(targetDir, out))
		if errors.Is(err, ErrSkipThisFile) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", out, err)
		}
		filenames = append(filenames, out)
	}

	return filenames, nil
}

// serviceFilename mirrors cliv0/service.go: account services go under
// cmd/account/<name>/<name>.go, everything else under cmd/workspace/.
func serviceFilename(s *ServiceJSON) string {
	name := s.TrimPrefix("account").KebabName
	if s.IsAccounts {
		return fmt.Sprintf("cmd/account/%s/%s.go", name, name)
	}
	return fmt.Sprintf("cmd/workspace/%s/%s.go", name, name)
}
