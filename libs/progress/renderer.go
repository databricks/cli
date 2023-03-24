package progress

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
)

var Red = color.New(color.FgRed).SprintFunc()
var Green = color.New(color.FgGreen).SprintFunc()
var Yellow = color.New(color.FgYellow).SprintFunc()

// Note, if something is printed to stdout, it might mess this rendering up. Lets
// see if we can detect it, and keep an accurate count of total log lines in console

type EventState string

const (
	EventStatePending   = EventState("pending")
	EventStateRunning   = EventState("running")
	EventStateCompleted = EventState("completed")
	EventStateFailed    = EventState("failed")
)

type Event struct {
	State       EventState
	Content     string
	IndentLevel int
}

type EventRenderer struct {
	mu          sync.Mutex
	events      []Event
	firstRender bool
	spinner     *spinner
	quit        chan int
}

// TODO: add support for generic output
type spinner struct {
	counter int
}

func (s *spinner) Step() {
	s.counter = (s.counter + 1) % 4
}

func (s *spinner) String() string {
	switch s.counter {
	case 0:
		return "◜"
	case 1:
		return "◝"
	case 2:
		return "◞"
	default:
		return "◟"
	}
}

// TODO: check if mutex works here or if we need some initialization
func NewEventRenderer() *EventRenderer {
	return &EventRenderer{
		events:      make([]Event, 0),
		firstRender: true,
		spinner:     &spinner{counter: 0},
		quit:        make(chan int),
	}
}

// TODO: write test for exclusive access
func (r *EventRenderer) AddEvent(state EventState, content string, indentLevel int) (id int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	event := Event{
		State:       state,
		Content:     content,
		IndentLevel: indentLevel,
	}
	r.events = append(r.events, event)
	return len(r.events) - 1
}

func (r *EventRenderer) UpdateState(id int, state EventState) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events[id].State = state
}

func (r *EventRenderer) UpdateContent(id int, content string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.events[id].Content = content
}

func cursorUp(v int) {
	fmt.Fprintf(os.Stderr, "\033[%dF", v)
}

func stateSymbol(state EventState) string {
	switch state {
	case EventStatePending:
		return Yellow("☉")
	case EventStateFailed:
		return Red("✗")
	case EventStateCompleted:
		return Green("✓")
	}
	return ""
}

func (r *EventRenderer) Start() {
	go func() {
		for {
			select {
			case <-r.quit:
				return
			default:
				r.Render()
				time.Sleep(200 * time.Millisecond)
			}
		}
	}()
}

func (r *EventRenderer) Close() {
	r.quit <- 1
}

// TODO: detect tty, and windows for ansi escapes
// TODO: add timpstamp (and other context stuff) for append only logger
func (r *EventRenderer) Render() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// TODO: comment
	if !r.firstRender {
		cursorUp(len(r.events) + 1)
	}
	r.firstRender = false

	r.spinner.Step()

	result := strings.Builder{}
	for _, event := range r.events {
		result.WriteString(strings.Repeat("    ", event.IndentLevel))
		if event.State == EventStateRunning {
			result.WriteString(r.spinner.String())
		} else {
			result.WriteString(stateSymbol(event.State))
		}
		result.WriteString(" ")
		result.WriteString(event.Content)
		result.WriteString("\n")
	}
	fmt.Fprintln(os.Stderr, result.String())
}
