// Package cligen regenerates the CLI command stubs (cmd/workspace/** and
// cmd/account/**) from the cli.json spec produced by genkit.
//
// It is the CLI-owned replacement for running genkit's cli_v0 generator against
// a universe checkout: it renders templates derived from genkit's cli_v0
// templates (templates/*.tmpl) against a model decoded from cli.json's
// "commands" block, then formats the output in-process. It has no dependency on
// genkit or any upstream spec at run time.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/databricks/cli/internal/clijson"
)

func main() {
	input := flag.String("input", ".codegen/cli.json", "path to the cli.json spec")
	output := flag.String("output", ".", "target directory (CLI repo root)")
	flag.Parse()

	files, err := Generate(*input, *output)
	if err != nil {
		log.Fatalf("cligen: %v", err)
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

// Generate reads the cli.json spec at jsonPath and writes the command stubs
// under targetDir. It returns the list of files written (relative to targetDir).
func Generate(jsonPath, targetDir string) ([]string, error) {
	doc, err := clijson.Parse(jsonPath)
	if err != nil {
		return nil, err
	}
	if doc.Commands == nil {
		return nil, fmt.Errorf("%s: missing \"commands\" block", jsonPath)
	}
	batch := fromContract(doc.Commands)
	if err := batch.Resolve(); err != nil {
		return nil, fmt.Errorf("%s: %w", jsonPath, err)
	}
	populateInputOnlyPaths(batch, doc.Schemas)

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
