package notebook

import "github.com/databricks/databricks-sdk-go/service/workspace"

type Extension string

const (
	ExtensionNone    Extension = ""
	ExtensionPython  Extension = ".py"
	ExtensionR       Extension = ".r"
	ExtensionScala   Extension = ".scala"
	ExtensionSql     Extension = ".sql"
	ExtensionJupyter Extension = ".ipynb"
)

var ExtensionToLanguage = map[Extension]workspace.Language{
	ExtensionPython: workspace.LanguagePython,
	ExtensionR:      workspace.LanguageR,
	ExtensionScala:  workspace.LanguageScala,
	ExtensionSql:    workspace.LanguageSql,

	// The platform supports all languages (Python, R, Scala, and SQL) for Jupyter notebooks.
}

func GetExtensionByLanguage(objectInfo *workspace.ObjectInfo) Extension {
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
