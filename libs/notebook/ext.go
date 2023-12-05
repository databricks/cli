package notebook

import "github.com/databricks/databricks-sdk-go/service/workspace"

func GetExtensionByLanguage(objectInfo *workspace.ObjectInfo) string {
	switch objectInfo.Language {
	case workspace.LanguagePython:
		return ".py"
	case workspace.LanguageR:
		return ".r"
	case workspace.LanguageScala:
		return ".scala"
	case workspace.LanguageSql:
		return ".sql"
	default:
		// Do not add any extension to the file name
		return ""
	}
}
