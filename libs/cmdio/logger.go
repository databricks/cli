package cmdio

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/flags"
	"github.com/manifoldco/promptui"
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

	mutex sync.Mutex
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

func AskSelect(ctx context.Context, question string, choices []string) (string, error) {
	logger, ok := FromContext(ctx)
	if !ok {
		logger = Default()
	}
	return logger.AskSelect(question, choices)
}

func splitAtLastNewLine(s string) (string, string) {
	// Split at the newline character
	if i := strings.LastIndex(s, "\n"); i != -1 {
		return s[:i+1], s[i+1:]
	}
	// Return the original string if no newline found
	return "", s
}

func (l *Logger) AskSelect(question string, choices []string) (string, error) {
	if l.Mode == flags.ModeJson {
		return "", errors.New("question prompts are not supported in json mode")
	}

	// Promptui does not support multiline prompts. So we split the question.
	first, last := splitAtLastNewLine(question)
	_, err := l.Writer.Write([]byte(first))
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
	}

	_, ans, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return ans, nil
}

func (l *Logger) Ask(question, defaultVal string) (string, error) {
	if l.Mode == flags.ModeJson {
		return "", errors.New("question prompts are not supported in json mode")
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
	_, _ = l.Writer.Write(b)
	_, _ = l.Writer.Write([]byte("\n"))
}

func (l *Logger) writeAppend(event Event) {
	_, _ = l.Writer.Write([]byte(event.String()))
	_, _ = l.Writer.Write([]byte("\n"))
}

func (l *Logger) writeInplace(event Event) {
	if l.isFirstEvent {
		// save cursor location
		_, _ = l.Writer.Write([]byte("\033[s"))
	}

	// move cursor to saved location
	_, _ = l.Writer.Write([]byte("\033[u"))

	// clear from cursor to end of screen
	_, _ = l.Writer.Write([]byte("\033[0J"))

	_, _ = l.Writer.Write([]byte(event.String()))
	_, _ = l.Writer.Write([]byte("\n"))
	l.isFirstEvent = false
}

func (l *Logger) Log(event Event) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
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
