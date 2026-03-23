package dresources

import "github.com/databricks/cli/libs/structs/fieldcopy"

type fieldCopyReporter interface {
	Report() string
}

var allFieldCopies []fieldCopyReporter

func registerCopy[Src, Dst any](c *fieldcopy.Copy[Src, Dst]) {
	c.Init()
	allFieldCopies = append(allFieldCopies, c)
}
