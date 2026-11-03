package cmdio

import (
	"context"
	"fmt"
	"io"
	"strings"
)

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

// Ask prompts the user with a question and returns the entered answer.
// If the user just presses enter, defaultVal is returned.
func Ask(ctx context.Context, question, defaultVal string) (string, error) {
	c := fromContext(ctx)

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

// AskCancellable is like Ask, but stops waiting for the answer when ctx is
// done and returns ctx.Err(). Reading the input cannot be interrupted, so the
// read keeps running in the background and its result is discarded. The
// caller must not read the input again after a cancelled call, because the
// abandoned read would consume the next line.
func AskCancellable(ctx context.Context, question, defaultVal string) (string, error) {
	type answer struct {
		ans string
		err error
	}
	answers := make(chan answer, 1)
	go func() {
		ans, err := Ask(ctx, question, defaultVal)
		answers <- answer{ans, err}
	}()

	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case a := <-answers:
		return a.ans, a.err
	}
}

// AskYesOrNo prompts the user with a question and returns true if the answer
// is "y" or "yes" (case-insensitive). Any other answer, including an empty
// one, returns false.
func AskYesOrNo(ctx context.Context, question string) (bool, error) {
	ans, err := Ask(ctx, question+" [y/N]", "")
	if err != nil {
		return false, err
	}
	ans = strings.ToLower(strings.TrimSpace(ans))
	return ans == "y" || ans == "yes", nil
}
