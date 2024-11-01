package notebook

import "github.com/databricks/databricks-sdk-go/service/workspace"

const (
	ExtensionNone    string = ""
	ExtensionPython  string = ".py"
	ExtensionR       string = ".r"
	ExtensionScala   string = ".scala"
	ExtensionSql     string = ".sql"
	ExtensionJupyter string = ".ipynb"
)

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
