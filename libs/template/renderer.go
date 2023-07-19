package template

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/logger"
	"golang.org/x/exp/slices"
)

type inMemoryFile struct {
	// Unix like path for the file (using '/' as the separator)
	path    string
	content []byte
}

// This structure renders any template files during project initialization
type renderer struct {
	ctx context.Context

	// A config that is the "dot" value available to any template being rendered.
	// Refer to https://pkg.go.dev/text/template for how templates can use
	// this "dot" value
	config map[string]any

	// A base template with helper functions and user defined template in the
	// library directory loaded. This is used as the base to compute any project
	// templates during file tree walk
	baseTemplate *template.Template

	// List of in memory files being generated from template.
	files        []*inMemoryFile
	skipPatterns []string

	templateFiler filer.Filer
	instanceFiler filer.Filer
}

func newRenderer(ctx context.Context, config map[string]any, libraryRoot, instanceRoot, templateRoot string) (*renderer, error) {
	// All user defined functions will be available inside library root
	libraryGlob := filepath.Join(libraryRoot, "*")

	// Initialize new template, with helper functions loaded
	tmpl := template.New("").Funcs(HelperFuncs)

	// Load files in the library to the template
	matches, err := filepath.Glob(libraryGlob)
	if err != nil {
		return nil, err
	}
	if len(matches) != 0 {
		tmpl, err = tmpl.ParseGlob(libraryGlob)
		if err != nil {
			return nil, err
		}
	}

	// create template filer
	templateFiler, err := filer.NewLocalClient(templateRoot)
	if err != nil {
		return nil, err
	}

	instanceFiler, err := filer.NewLocalClient(instanceRoot)
	if err != nil {
		return nil, err
	}

	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("action", "initialize-template"))

	return &renderer{
		config:        config,
		baseTemplate:  tmpl,
		files:         make([]*inMemoryFile, 0),
		templateFiler: templateFiler,
		ctx:           ctx,
		skipPatterns:  make([]string, 0),
		instanceFiler: instanceFiler,
	}, nil
}

// Executes the template by applying config on it. Returns the materialized template
// as a string
func (r *renderer) executeTemplate(templateDefinition string) (string, error) {
	// Create copy of base template so as to not overwrite it
	tmpl, err := r.baseTemplate.Clone()
	if err != nil {
		return "", err
	}

	// Parse the template text
	tmpl, err = tmpl.Parse(templateDefinition)
	if err != nil {
		return "", err
	}

	// Execute template and get result
	result := strings.Builder{}
	err = tmpl.Execute(&result, r.config)
	if err != nil {
		return "", err
	}
	return result.String(), nil
}

func (r *renderer) computeFile(relPathTemplate string) (*inMemoryFile, error) {
	// read template file contents
	templateReader, err := r.templateFiler.Read(r.ctx, relPathTemplate)
	if err != nil {
		return nil, err
	}
	contentTemplate, err := io.ReadAll(templateReader)
	if err != nil {
		return nil, err
	}

	// execute the contents of the file as a template
	content, err := r.executeTemplate(string(contentTemplate))
	// Capture errors caused by the "fail" helper function
	if target := (&ErrFail{}); errors.As(err, target) {
		return nil, target
	}
	if err != nil {
		return nil, fmt.Errorf("failed to compute file content for %s. %w", relPathTemplate, err)
	}

	// Execute relative path template to get materialized path for the file
	relPath, err := r.executeTemplate(relPathTemplate)
	if err != nil {
		return nil, err
	}

	return &inMemoryFile{
		path:    relPath,
		content: []byte(content),
	}, nil
}

func (r *renderer) walk() error {
	directories := []string{"."}
	var currentDirectory string

	for len(directories) > 0 {
		currentDirectory, directories = directories[0], directories[1:]

		// Skip current directory if it matches any of accumulated skip patterns
		instanceDirectory, err := r.executeTemplate(currentDirectory)
		if err != nil {
			return err
		}
		isSkipped, err := r.isSkipped(instanceDirectory)
		if err != nil {
			return err
		}
		if isSkipped {
			logger.Infof(r.ctx, "skipping walking directory: %s", instanceDirectory)
			continue
		}

		// Add skip function, which accumulates skip patterns relative to current
		// directory
		r.baseTemplate.Funcs(template.FuncMap{
			"skip": func(relPattern string) error {
				// patterns are specified relative to current directory of the file
				// {{skip}} function is called from
				pattern := path.Join(currentDirectory, relPattern)
				if !slices.Contains(r.skipPatterns, pattern) {
					logger.Infof(r.ctx, "adding skip pattern: %s", pattern)
					r.skipPatterns = append(r.skipPatterns, pattern)
				}
				return nil
			},
		})

		// Process all entries in current directory
		//
		// 1. For files: the templates in the file name and content are executed, and
		//     a in memory representation of the file is generated
		//
		// 2. For directories: They are appended to a slice, which acts as a queue
		//     allowing BFS traversal of the template file tree
		entries, err := r.templateFiler.ReadDir(r.ctx, currentDirectory)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				// Add to slice, for BFS traversal
				directories = append(directories, path.Join(currentDirectory, entry.Name()))
				continue
			}

			// Generate in memory representation of file
			f, err := r.computeFile(path.Join(currentDirectory, entry.Name()))
			if err != nil {
				return err
			}
			logger.Infof(r.ctx, "added file to in memory file tree: %s", f.path)
			r.files = append(r.files, f)
		}

	}
	return nil
}

func (r *renderer) persistToDisk() error {
	for _, file := range r.files {
		isSkipped, err := r.isSkipped(file.path)
		if err != nil {
			return err
		}
		if isSkipped {
			log.Infof(r.ctx, "skipping file: %s", file.path)
			continue
		}
		err = r.instanceFiler.Write(r.ctx, file.path, bytes.NewReader(file.content), filer.CreateParentDirectories)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *renderer) isSkipped(filePath string) (bool, error) {
	for _, pattern := range r.skipPatterns {
		isMatch, err := path.Match(pattern, filePath)
		if err != nil {
			return false, err
		}
		if isMatch {
			return true, nil
		}
	}
	return false, nil
}
