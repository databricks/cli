package diag

type Severity int

const (
	Error Severity = iota
	Warning
	Info
	Recommendation
)
