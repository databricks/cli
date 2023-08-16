package cmdio

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/databricks/cli/libs/flags"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// This is the interface for all io interactions with a user
type Logger struct {
	// Mode for the logger. One of (append, inplace, json).
	Mode flags.ProgressLogFormat

	// Input stream (eg. stdin). Answers to questions prompted using the Ask() method
	// are read from here
	Reader bufio.Reader

	// Output stream where the logger writes to
	Writer io.Writer

	// If true, indicates no events have been printed by the logger yet. Used
	// by inplace logging for formatting
	isFirstEvent bool
}

func NewLogger(mode flags.ProgressLogFormat) *Logger {
	return &Logger{
		Mode:         mode,
		Writer:       os.Stderr,
		Reader:       *bufio.NewReader(os.Stdin),
		isFirstEvent: true,
	}
}

func Default() *Logger {
	return &Logger{
		Mode:         flags.ModeAppend,
		Writer:       os.Stderr,
		Reader:       *bufio.NewReader(os.Stdin),
		isFirstEvent: true,
	}
}

func Log(ctx context.Context, event Event) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}
	logger.Log(event)
}

func LogString(ctx context.Context, message string) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}
	logger.Log(&MessageEvent{
		Message: message,
	})
}

func LogError(ctx context.Context, err error) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}
	logger.Log(&ErrorEvent{
		Error: err.Error(),
	})
}

func Ask(ctx context.Context, question, defaultVal string) (string, error) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}
	return logger.Ask(question, defaultVal)
}

func AskYesOrNo(ctx context.Context, question string) (bool, error) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}

	// Add acceptable answers to the question prompt.
	question += ` [y/n]`

	// Ask the question
	ans, err := logger.Ask(question, "")
	if err != nil {
		return false, err
	}

	if ans == "y" {
		return true, nil
	}
	return false, nil
}

func AskChoice(ctx context.Context, question string, defaultVal string, choices []string) (string, error) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}

	// Find index of the default value
	defaultIndex := ""
	for i, v := range choices {
		if v == defaultVal {
			defaultIndex = fmt.Sprint(i + 1)
			break
		}
	}

	// If default value is not present, return error
	if defaultIndex == "" {
		return "", fmt.Errorf("failed to find default value %q among choices: %#v", defaultVal, choices)
	}

	indexToChoice := make(map[string]string, 0)
	question += ":\n"
	for index, choice := range choices {
		// Map choices against a string representation of their indices.
		// This helps resolve the choice corresponding to the index the user enters.
		choiceIndex := fmt.Sprint(index + 1)
		indexToChoice[choiceIndex] = choice

		// Add this choice as a option in the prompt text.
		question += fmt.Sprintf("%s. %s\n", choiceIndex, choice)

	}

	// Add text informing user of the list of valid options to choose from
	question += fmt.Sprintf("Choose from %s", strings.Join(maps.Keys(indexToChoice), ", "))

	// prompt the user.
	ans, err := logger.Ask(question, defaultIndex)
	if err != nil {
		return "", err
	}

	choice, ok := indexToChoice[ans]
	if !ok {
		expectedOptions := maps.Keys(indexToChoice)
		slices.Sort(expectedOptions)
		return "", fmt.Errorf("expected one of %s. Got: %s", strings.Join(expectedOptions, ", "), ans)
	}

	return choice, nil
}

func (l *Logger) Ask(question string, defaultVal string) (string, error) {
	if l.Mode == flags.ModeJson {
		return "", fmt.Errorf("question prompts are not supported in json mode")
	}

	// Add default value to question prompt.
	if defaultVal != "" {
		question += fmt.Sprintf(` [%s]`, defaultVal)
	}
	question += `: `

	// print prompt
	_, err := l.Writer.Write([]byte(question))
	if err != nil {
		return "", err
	}

	// read user input. Trim new line characters
	ans, err := l.Reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	ans = strings.Trim(ans, "\n\r")

	// Return default value if user just presses enter
	if ans == "" {
		return defaultVal, nil
	}
	return ans, nil
}

func (l *Logger) writeJson(event Event) {
	b, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		// we panic because there we cannot catch this in jobs.RunNowAndWait
		panic(err)
	}
	l.Writer.Write([]byte(b))
	l.Writer.Write([]byte("\n"))
}

func (l *Logger) writeAppend(event Event) {
	l.Writer.Write([]byte(event.String()))
	l.Writer.Write([]byte("\n"))
}

func (l *Logger) writeInplace(event Event) {
	if l.isFirstEvent {
		// save cursor location
		l.Writer.Write([]byte("\033[s"))
	}

	// move cursor to saved location
	l.Writer.Write([]byte("\033[u"))

	// clear from cursor to end of screen
	l.Writer.Write([]byte("\033[0J"))

	l.Writer.Write([]byte(event.String()))
	l.Writer.Write([]byte("\n"))
	l.isFirstEvent = false
}

func (l *Logger) Log(event Event) {
	switch l.Mode {
	case flags.ModeInplace:
		if event.IsInplaceSupported() {
			l.writeInplace(event)
		} else {
			l.writeAppend(event)
		}

	case flags.ModeJson:
		l.writeJson(event)

	case flags.ModeAppend:
		l.writeAppend(event)

	default:
		// we panic because errors are not captured in some log sides like
		// jobs.RunNowAndWait
		panic("unknown progress logger mode: " + l.Mode.String())
	}

}
