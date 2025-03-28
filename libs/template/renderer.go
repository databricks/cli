package template

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"sort"
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

	// [fs.FS] that holds the template's file tree.
	srcFS fs.FS
}

func newRenderer(
	ctx context.Context,
	config map[string]any,
	helpers template.FuncMap,
	templateFS fs.FS,
	templateDir string,
	libraryDir string,
) (*renderer, error) {
	// Initialize new template, with helper functions loaded
	tmpl := template.New("").Funcs(helpers)

	// Find user-defined templates in the library directory
	matches, err := fs.Glob(templateFS, path.Join(libraryDir, "*"))
	if err != nil {
		return nil, err
	}

	// Parse user-defined templates.
	// Note: we do not call [ParseFS] with the glob directly because
	// it returns an error if no files match the pattern.
	if len(matches) != 0 {
		tmpl, err = tmpl.ParseFS(templateFS, matches...)
		if err != nil {
			return nil, err
		}
	}

	srcFS, err := fs.Sub(templateFS, path.Clean(templateDir))
	if err != nil {
		return nil, err
	}

	ctx = log.NewContext(ctx, log.GetLogger(ctx).With("action", "initialize-template"))

	return &renderer{
		ctx:          ctx,
		config:       config,
		baseTemplate: tmpl,
		files:        make([]file, 0),
		skipPatterns: make([]string, 0),
		srcFS:        srcFS,
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

	// The template execution will error instead of printing <no value> on unknown
	// map keys if the "missingkey=error" option is set.
	// We do this here instead of doing this once for r.baseTemplate because
	// the Template.Clone() method does not clone options.
	tmpl = tmpl.Option("missingkey=error")

	// Parse the template text
	tmpl, err = tmpl.Parse(templateDefinition)
	if err != nil {
		return "", fmt.Errorf("error in %s: %w", templateDefinition, err)
	}

	// Execute template and get result
	result := strings.Builder{}
	err = tmpl.Execute(&result, r.config)
	if err != nil {
		// Parse and return a more readable error for missing values that are used
		// by the template definition but are not provided in the passed config.
		target := &template.ExecError{}
		if errors.As(err, target) {
			captureRegex := regexp.MustCompile(`map has no entry for key "(.*)"`)
			matches := captureRegex.FindStringSubmatch(target.Err.Error())
			if len(matches) != 2 {
				return "", err
			}
			return "", template.ExecError{
				Name: target.Name,
				Err:  fmt.Errorf("variable %q not defined", matches[1]),
			}
		}
		return "", err
	}
	return result.String(), nil
}

func (r *renderer) computeFile(relPathTemplate string) (file, error) {
	// read file permissions
	info, err := fs.Stat(r.srcFS, relPathTemplate)
	if err != nil {
		return nil, err
	}
	perm := info.Mode().Perm()

	// Always include the write bit for the owner of the file.
	// It does not make sense to have a file that is not writable by the owner.
	perm |= 0o200

	// Execute relative path template to get destination path for the file
	relPath, err := r.executeTemplate(relPathTemplate)
	if err != nil {
		return nil, err
	}

	// If file name does not specify the `.tmpl` extension, then it is copied
	// over as is, without treating it as a template
	if !strings.HasSuffix(relPathTemplate, templateExtension) {
		return &copyFile{
			perm:    perm,
			relPath: relPath,
			srcFS:   r.srcFS,
			srcPath: relPathTemplate,
		}, nil
	} else {
		// Trim the .tmpl suffix from file name, if specified in the template
		// path
		relPath = strings.TrimSuffix(relPath, templateExtension)
	}

	// read template file's content
	templateReader, err := r.srcFS.Open(relPathTemplate)
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
		perm:    perm,
		relPath: relPath,
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
			"skip": func(relPattern string) (string, error) {
				// patterns are specified relative to current directory of the file
				// the {{skip}} function is called from.
				patternRaw := path.Join(currentDirectory, relPattern)
				pattern, err := r.executeTemplate(patternRaw)
				if err != nil {
					return "", err
				}

				if !slices.Contains(r.skipPatterns, pattern) {
					logger.Infof(r.ctx, "adding skip pattern: %s", pattern)
					r.skipPatterns = append(r.skipPatterns, pattern)
				}
				// return empty string will print nothing at function call site
				// when executing the template
				return "", nil
			},
		})

		// Process all entries in current directory
		//
		// 1. For files: the templates in the file name and content are executed, and
		//     a in memory representation of the file is generated
		//
		// 2. For directories: They are appended to a slice, which acts as a queue
		//     allowing BFS traversal of the template file tree
		entries, err := fs.ReadDir(r.srcFS, currentDirectory)
		if err != nil {
			return err
		}
		// Sort by name to ensure deterministic ordering
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name() < entries[j].Name()
		})
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
			logger.Infof(r.ctx, "added file to list of possible project files: %s", f.RelPath())
			r.files = append(r.files, f)
		}

	}
	return nil
}

func (r *renderer) persistToDisk(ctx context.Context, out filer.Filer) error {
	// Accumulate files which we will persist, skipping files whose path matches
	// any of the skip patterns
	var filesToPersist []file
	for _, file := range r.files {
		match, err := isSkipped(file.RelPath(), r.skipPatterns)
		if err != nil {
			return err
		}
		if match {
			log.Infof(r.ctx, "skipping file: %s", file.RelPath())
			continue
		}
		filesToPersist = append(filesToPersist, file)
	}

	// Assert no conflicting files exist
	for _, file := range filesToPersist {
		path := file.RelPath()
		_, err := out.Stat(ctx, path)
		if err == nil {
			return fmt.Errorf("failed to initialize template, one or more files already exist: %s", path)
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("error while verifying file %s does not already exist: %w", path, err)
		}
	}

	// Persist files to disk
	for _, file := range filesToPersist {
		err := file.Write(ctx, out)
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
