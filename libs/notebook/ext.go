package notebook

import (
	"path"
	"strings"

	"github.com/databricks/databricks-sdk-go/service/workspace"
)

const (
	ExtensionNone    string = ""
	ExtensionPython  string = ".py"
	ExtensionR       string = ".r"
	ExtensionScala   string = ".scala"
	ExtensionSql     string = ".sql"
	ExtensionJupyter string = ".ipynb"
	// ExtensionDesigner is the compound suffix for Lakeflow Designer files.
	// Unlike other notebook types, designer files keep this full suffix when
	// imported into the workspace.
	ExtensionDesigner string = ".designer.ipynb"
)

// StripExtension returns the workspace path for a local notebook file.
// Designer files keep their full ".designer.ipynb" suffix in the workspace;
// other notebook types lose their extension on import.
func StripExtension(name string) string {
	if strings.HasSuffix(name, ExtensionDesigner) {
		return name
	}
	return strings.TrimSuffix(name, path.Ext(name))
}

// Extensions lists all notebook file extensions.
var Extensions = []string{
	ExtensionPython,
	ExtensionR,
	ExtensionScala,
	ExtensionSql,
	ExtensionJupyter,
}

var ExtensionToLanguage = map[string]workspace.Language{
	ExtensionPython: workspace.LanguagePython,
	ExtensionR:      workspace.LanguageR,
	ExtensionScala:  workspace.LanguageScala,
	ExtensionSql:    workspace.LanguageSql,

	// The platform supports all languages (Python, R, Scala, and SQL) for Jupyter notebooks.
}

func GetExtensionByLanguage(objectInfo *workspace.ObjectInfo) string {
	if objectInfo.ObjectType != workspace.ObjectTypeNotebook {
		return ExtensionNone
	}

	switch objectInfo.Language {
	case workspace.LanguagePython:
		return ExtensionPython
	case workspace.LanguageR:
		return ExtensionR
	case workspace.LanguageScala:
		return ExtensionScala
	case workspace.LanguageSql:
		return ExtensionSql
	default:
		// Do not add any extension to the file name
		return ExtensionNone
	}
}
