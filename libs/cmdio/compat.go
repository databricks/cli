package cmdio

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/manifoldco/promptui"
)

/*
Temporary compatibility layer for the progress logger interfaces.
*/

// Log is a compatibility layer for the progress logger interfaces.
// It writes the string representation of the stringer to the error writer.
// Suppressed in quiet mode.
func Log(ctx context.Context, str fmt.Stringer) {
	LogString(ctx, str.String())
}

// LogString is a compatibility layer for the progress logger interfaces.
// It writes the string to the error writer.
// Suppressed in quiet mode.
func LogString(ctx context.Context, str string) {
	c := fromContext(ctx)
	// Quiet mode suppresses informational output (progress, status messages)
	// while still allowing Render() to write structured data to stdout.
	if c.capabilities.quiet {
		return
	}
	_, _ = io.WriteString(c.err, str+"\n")
}

// readLine reads a line from the reader and returns it without the trailing newline characters.
// It is unbuffered because cmdio's stdin is also unbuffered.
// If we were to add a [bufio.Reader] to the mix, we would need to update the other uses of the reader.
// Once cmdio's stdio is made to be buffered, this function can be removed.
func readLine(r io.Reader) (string, error) {
	var b strings.Builder
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if buf[0] == '\n' {
				break
			}
			if buf[0] != '\r' {
				b.WriteByte(buf[0])
			}
		}
		if err != nil {
			if b.Len() == 0 {
				return "", err
			}
			break
		}
	}
	return b.String(), nil
}

// Ask is a compatibility layer for the progress logger interfaces.
// It prompts the user with a question and returns the answer.
// When --no-input is set, returns the default value if one is provided,
// otherwise returns ErrNoInput.
func Ask(ctx context.Context, question, defaultVal string) (string, error) {
	c := fromContext(ctx)

	if c.capabilities.noInput {
		if defaultVal != "" {
			return defaultVal, nil
		}
		return "", ErrNoInput
	}

	// Add default value to question prompt.
	if defaultVal != "" {
		question += fmt.Sprintf(` [%s]`, defaultVal)
	}
	question += `: `

	// Print prompt.
	_, err := io.WriteString(c.err, question)
	if err != nil {
		return "", err
	}

	// Read user input. Trim new line characters.
	ans, err := readLine(c.in)
	if err != nil {
		return "", err
	}

	// Return default value if user just presses enter.
	if ans == "" {
		return defaultVal, nil
	}

	return ans, nil
}

// AskYesOrNo is a compatibility layer for the progress logger interfaces.
// It prompts the user with a question and returns the answer.
// Precedence: --yes (returns true) > --no-input (returns ErrNoInput) > normal prompting.
func AskYesOrNo(ctx context.Context, question string) (bool, error) {
	c := fromContext(ctx)

	if c.capabilities.yes {
		return true, nil
	}
	if c.capabilities.noInput {
		return false, ErrNoInput
	}

	ans, err := Ask(ctx, question+" [y/n]", "")
	if err != nil {
		return false, err
	}
	return ans == "y", nil
}

func splitAtLastNewLine(s string) (string, string) {
	// Split at the newline character
	if i := strings.LastIndex(s, "\n"); i != -1 {
		return s[:i+1], s[i+1:]
	}
	// Return the original string if no newline found
	return "", s
}

// AskSelect is a compatibility layer for the progress logger interfaces.
// It prompts the user with a question and returns the answer.
// Returns ErrNoInput when --no-input is set.
func AskSelect(ctx context.Context, question string, choices []string) (string, error) {
	c := fromContext(ctx)

	if c.capabilities.noInput {
		return "", ErrNoInput
	}

	// Promptui does not support multiline prompts. So we split the question.
	first, last := splitAtLastNewLine(question)
	_, err := io.WriteString(c.err, first)
	if err != nil {
		return "", err
	}

	prompt := promptui.Select{
		Label:    last,
		Items:    choices,
		HideHelp: true,
		Templates: &promptui.SelectTemplates{
			Label:    "{{.}}: ",
			Selected: last + ": {{.}}",
		},
		Stdin:  c.promptStdin(),
		Stdout: nopWriteCloser{c.err},
	}

	_, ans, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return ans, nil
}
