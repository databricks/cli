package libraries

import "github.com/databricks/databricks-sdk-go/service/compute"

func libraryPath(library *compute.Library) string {
	if library.Whl != "" {
		return library.Whl
	}
	if library.Jar != "" {
		return library.Jar
	}
	if library.Egg != "" {
		return library.Egg
	}
	return ""
}
