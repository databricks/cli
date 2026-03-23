package dresources

import "github.com/databricks/cli/libs/structs/fieldcopy"

type fieldCopyReporter interface {
	Report() string
}

var allFieldCopies []fieldCopyReporter

func newCopy[Src, Dst any]() *fieldcopy.Copy[Src, Dst] {
	c := &fieldcopy.Copy[Src, Dst]{}
	c.Init()
	allFieldCopies = append(allFieldCopies, c)
	return c
}
