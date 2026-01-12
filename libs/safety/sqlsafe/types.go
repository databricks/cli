package sqlsafe

import "fmt"

// Position identifies a 1-indexed line and column within the original SQL text.
type Position struct {
	Line   int
	Column int
}

// Statement represents a parsed SQL statement without comments.
type Statement struct {
	// Text contains the sanitized SQL statement with comments removed and
	// surrounding whitespace trimmed. The text never includes a trailing
	// semicolon.
	Text string

	// OriginalText contains the statement exactly as written in the input,
	// minus the trailing semicolon. Comments and original whitespace inside
	// the statement are preserved.
	OriginalText string

	// FirstKeyword is the uppercase representation of the first significant
	// token in the statement. It is empty when the statement does not
	// contain any non-whitespace tokens (e.g. an empty statement).
	FirstKeyword string

	// FirstKeywordPos marks the location of FirstKeyword in the original
	// SQL text. Both the line and column are 1-indexed.
	FirstKeywordPos Position
}

// Violation reports that a statement violates the configured safety policy.
type Violation struct {
	StatementIndex int
	Keyword        string
	Position       Position
	Reason         string
}

func (v *Violation) Error() string {
	if v.Reason != "" {
		return v.Reason
	}
	line := v.Position.Line
	column := v.Position.Column
	if line <= 0 {
		line = 1
	}
	if column <= 0 {
		column = 1
	}
	return fmt.Sprintf("blocked SQL statement at line %d, column %d", line, column)
}
