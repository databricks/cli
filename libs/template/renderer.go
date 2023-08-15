package template

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/logger"
)

const templateExtension = ".tmpl"

// Renders a databricks template as a project
type renderer struct {
	ctx context.Context

	// A config that is the "dot" value available to any template being rendered.
	// Refer to https://pkg.go.dev/text/template for how templates can use
	// this "dot" value
	config map[string]any

	// A base template with helper functions and user defined templates in the
	// library directory loaded. This is cloned for each project template computation
	// during file tree walk
	baseTemplate *template.Template

	// List of in memory files generated from template
	files []file

	// Glob patterns for files and directories to skip. There are three possible
	// outcomes for skip:
	//
	// 1. File is not generated. This happens if one of the file's parent directories
	// match a glob pattern
	//
	// 2. File is generated but not persisted to disk. This happens if the file itself
	// matches a glob pattern, but none of it's parents match a glob pattern from the list
	//
	// 3. File is persisted to disk. This happens if the file and it's parent directories
	// do not match any glob patterns from this list
	skipPatterns []string

	// Filer rooted at template root. The file tree from this root is walked to
	// generate the project
	templateFiler filer.Filer

	// Root directory for the project instantiated from the template
	instanceRoot string
}

func newRenderer(ctx context.Context, config map[string]any, templateRoot, libraryRoot, instanceRoot string) (*renderer, error) {
	// Initialize new template, with helper functions loaded
	tmpl := template.New("").Funcs(helperFuncs)

	// Load user defined associated templates from the library root
	libraryGlob := filepath.Join(libraryRoot, "*")
	matches, err := filepath.Glob(libraryGlob)
	if err != nil {
		return nil, err
	}
	if len(matches) != 0 {
		tmpl, err = tmpl.ParseFiles(matches...)
		if err != nil {
			return nil, err
		}
	}

	templateFiler, err := filer.NewLocalClient(templateRoot)
	if err != nil {
		return nil, err
	}

	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("action", "initialize-template"))

	return &renderer{
		ctx:           ctx,
		config:        config,
		baseTemplate:  tmpl,
		files:         make([]file, 0),
		skipPatterns:  make([]string, 0),
		templateFiler: templateFiler,
		instanceRoot:  instanceRoot,
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

func (r *renderer) computeFile(relPathTemplate string) (file, error) {
	// read file permissions
	info, err := r.templateFiler.Stat(r.ctx, relPathTemplate)
	if err != nil {
		return nil, err
	}
	perm := info.Mode().Perm()

	// Execute relative path template to get destination path for the file
	relPath, err := r.executeTemplate(relPathTemplate)
	if err != nil {
		return nil, err
	}

	// If file name does not specify the `.tmpl` extension, then it is copied
	// over as is, without treating it as a template
	if !strings.HasSuffix(relPathTemplate, templateExtension) {
		return &copyFile{
			dstPath: &destinationPath{
				root:    r.instanceRoot,
				relPath: relPath,
			},
			perm:     perm,
			ctx:      r.ctx,
			srcPath:  relPathTemplate,
			srcFiler: r.templateFiler,
		}, nil
	} else {
		// Trim the .tmpl suffix from file name, if specified in the template
		// path
		relPath = strings.TrimSuffix(relPath, templateExtension)
	}

	// read template file's content
	templateReader, err := r.templateFiler.Read(r.ctx, relPathTemplate)
	if err != nil {
		return nil, err
	}
	defer templateReader.Close()

	// execute the contents of the file as a template
	contentTemplate, err := io.ReadAll(templateReader)
	if err != nil {
		return nil, err
	}
	content, err := r.executeTemplate(string(contentTemplate))
	// Capture errors caused by the "fail" helper function
	if target := (&ErrFail{}); errors.As(err, target) {
		return nil, target
	}
	if err != nil {
		return nil, fmt.Errorf("failed to compute file content for %s. %w", relPathTemplate, err)
	}

	return &inMemoryFile{
		dstPath: &destinationPath{
			root:    r.instanceRoot,
			relPath: relPath,
		},
		perm:    perm,
		content: []byte(content),
	}, nil
}

// This function walks the template file tree to generate an in memory representation
// of a project.
//
// During file tree walk, in the current directory, we would like to determine
// all possible {{skip}} function calls before we process any of the directories
// so that we can skip them eagerly if needed. That is in the current working directory
// we would like to process all files before we process any of the directories.
//
// This is not possible using the std library WalkDir which processes the files in
// lexical order which is why this function implements BFS.
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
		match, err := isSkipped(instanceDirectory, r.skipPatterns)
		if err != nil {
			return err
		}
		if match {
			logger.Infof(r.ctx, "skipping directory: %s", instanceDirectory)
			continue
		}

		// Add skip function, which accumulates skip patterns relative to current
		// directory
		r.baseTemplate.Funcs(template.FuncMap{
			"skip": func(relPattern string) string {
				// patterns are specified relative to current directory of the file
				// the {{skip}} function is called from.
				pattern := path.Join(currentDirectory, relPattern)
				if !slices.Contains(r.skipPatterns, pattern) {
					logger.Infof(r.ctx, "adding skip pattern: %s", pattern)
					r.skipPatterns = append(r.skipPatterns, pattern)
				}
				// return empty string will print nothing at function call site
				// when executing the template
				return ""
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
			logger.Infof(r.ctx, "added file to list of possible project files: %s", f.DstPath().relPath)
			r.files = append(r.files, f)
		}

	}
	return nil
}

func (r *renderer) persistToDisk() error {
	// Accumulate files which we will persist, skipping files whose path matches
	// any of the skip patterns
	filesToPersist := make([]file, 0)
	for _, file := range r.files {
		match, err := isSkipped(file.DstPath().relPath, r.skipPatterns)
		if err != nil {
			return err
		}
		if match {
			log.Infof(r.ctx, "skipping file: %s", file.DstPath())
			continue
		}
		filesToPersist = append(filesToPersist, file)
	}

	// Assert no conflicting files exist
	for _, file := range filesToPersist {
		path := file.DstPath().absPath()
		_, err := os.Stat(path)
		if err == nil {
			return fmt.Errorf("failed to persist to disk, conflict with existing file: %s", path)
		}
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("error while verifying file %s does not already exist: %w", path, err)
		}
	}

	// Persist files to disk
	for _, file := range filesToPersist {
		err := file.PersistToDisk()
		if err != nil {
			return err
		}
	}
	return nil
}

func isSkipped(filePath string, patterns []string) (bool, error) {
	for _, pattern := range patterns {
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
