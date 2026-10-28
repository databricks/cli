package notebook

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

type jupyterDatabricksMetadata struct {
	Language     string `json:"language"`
	NotebookName string `json:"notebookName"`
}

// See https://nbformat.readthedocs.io/en/latest/format_description.html#top-level-structure.
type jupyter struct {
	Cells         []json.RawMessage          `json:"cells,omitempty"`
	Metadata      map[string]json.RawMessage `json:"metadata,omitempty"`
	NbFormatMajor int                        `json:"nbformat"`
	NbFormatMinor int                        `json:"nbformat_minor"`
}

// resolveLanguage looks at Databricks specific metadata to figure out the language of the notebook.
func resolveLanguage(nb *jupyter) workspace.Language {
	if nb.Metadata == nil {
		return ""
	}

	raw, ok := nb.Metadata["application/vnd.databricks.v1+notebook"]
	if !ok {
		return ""
	}

	var metadata jupyterDatabricksMetadata
	err := json.Unmarshal(raw, &metadata)
	if err != nil {
		// Fine to swallow error. The file must be malformed.
		return ""
	}

	switch metadata.Language {
	case "python":
		return workspace.LanguagePython
	case "r":
		return workspace.LanguageR
	case "scala":
		return workspace.LanguageScala
	case "sql":
		return workspace.LanguageSql
	default:
		return ""
	}
}

// DetectJupyter returns whether the file at path is a valid Jupyter notebook.
// We assume it is valid if we can read it as JSON and see a couple expected fields.
// If we cannot, importing into the workspace will always fail, so we also return an error.
func DetectJupyterWithFS(fsys fs.FS, name string) (notebook bool, language workspace.Language, err error) {
	f, err := fsys.Open(name)
	if err != nil {
		return false, "", err
	}

	defer f.Close()

	var nb jupyter
	dec := json.NewDecoder(f)
	err = dec.Decode(&nb)
	if err != nil {
		return false, "", fmt.Errorf("%s: error loading Jupyter notebook file: %w", name, err)
	}

	// Not a Jupyter notebook if the cells or metadata fields aren't defined.
	if nb.Cells == nil || nb.Metadata == nil {
		return false, "", fmt.Errorf("%s: invalid Jupyter notebook file", name)
	}

	// Major version must be at least 4.
	if nb.NbFormatMajor < 4 {
		return false, "", fmt.Errorf("%s: unsupported Jupyter notebook version: %d", name, nb.NbFormatMajor)
	}

	return true, resolveLanguage(&nb), nil
}

// DetectJupyter calls DetectJupyterWithFS with the local filesystem.
// The name argument may be a local relative path or a local absolute path.
func DetectJupyter(name string) (notebook bool, language workspace.Language, err error) {
	d := filepath.ToSlash(filepath.Dir(name))
	b := filepath.Base(name)
	return DetectJupyterWithFS(os.DirFS(d), b)
}
