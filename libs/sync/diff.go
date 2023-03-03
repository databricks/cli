package sync

type diff struct {
	put    []string
	delete []string
}

func (d diff) IsEmpty() bool {
	return len(d.put) == 0 && len(d.delete) == 0
}
