package cmdio

import (
	"context"
	"fmt"
	"io"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Glyphs drawn into the prompt's rendered output. cursorBlock stands in for
// the (hidden) OS cursor at the current edit position.
const (
	cursorBlock  = "█"
	glyphValid   = "✔"
	glyphInvalid = "✗"
)

// PromptOptions configures a single-line text prompt shown by [RunPrompt].
type PromptOptions struct {
	// Label is shown before the input field. Required.
	Label string

	// Mask, when non-zero, replaces typed characters with the given rune
	// (use '*' for password-style input).
	Mask rune

	// HideEntered clears the prompt line after submission so the entered
	// value is not left behind in scrollback. Used by [Secret].
	HideEntered bool

	// Validate, when set, is called on every keystroke. While it returns a
	// non-nil error the leading glyph flips from "✔" to "✗" and Enter is
	// inert; pressing Enter while invalid surfaces the returned error
	// below the prompt until the next edit.
	Validate func(input string) error
}

// RunPrompt shows a single-line text prompt and returns the entered value.
// Returns an error without prompting when the terminal does not support it.
func RunPrompt(ctx context.Context, opts PromptOptions) (string, error) {
	c := fromContext(ctx)
	if !c.capabilities.SupportsPrompt() {
		return "", fmt.Errorf("expected to have %s", opts.Label)
	}
	return c.runPromptModel(newPromptModel(opts))
}

// Secret prompts the user for a value while masking input with '*' and
// clearing the prompt line on submission so the masked value isn't left
// behind in scrollback.
func Secret(ctx context.Context, label string) (string, error) {
	return RunPrompt(ctx, PromptOptions{
		Label:       label,
		Mask:        '*',
		HideEntered: true,
	})
}

type promptModel struct {
	label       string
	mask        rune
	hideEntered bool
	validate    func(string) error

	// runes holds the editable input as a slice of runes so cursor positions
	// remain valid for multibyte characters.
	runes  []rune
	cursor int

	cancelled bool
	deleted   bool
	submitted bool

	// submitErr is the error from the last failed Enter attempt. Rendered
	// below the prompt and cleared on the next edit.
	submitErr error
}

func newPromptModel(opts PromptOptions) *promptModel {
	return &promptModel{
		label:       opts.Label,
		mask:        opts.Mask,
		hideEntered: opts.HideEntered,
		validate:    opts.Validate,
	}
}

func (m *promptModel) value() string {
	return string(m.runes)
}

// glyph returns the leading status indicator: bold-green ✔ when valid,
// bold-red ✗ when validate rejects the buffer.
func (m *promptModel) glyph() string {
	if m.validate != nil && m.validate(m.value()) != nil {
		return ansiBold + ansiRed + glyphInvalid + ansiReset
	}
	return ansiBold + ansiGreen + glyphValid + ansiReset
}

func (m *promptModel) Init() tea.Cmd { return nil }

func (m *promptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	switch key.Type {
	case tea.KeyCtrlC:
		m.cancelled = true
		return m, tea.Quit

	case tea.KeyEnter, tea.KeyCtrlJ:
		// Enter sends CR, Ctrl+J sends LF; treat both as submit.
		if m.validate != nil {
			if err := m.validate(m.value()); err != nil {
				m.submitErr = err
				return m, nil
			}
		}
		m.submitted = true
		return m, tea.Quit

	case tea.KeyDelete, tea.KeyCtrlD:
		// Delete and Ctrl+D both exit the prompt with EOF, even on a
		// non-empty buffer.
		m.deleted = true
		return m, tea.Quit

	case tea.KeyLeft, tea.KeyCtrlB:
		if m.cursor > 0 {
			m.cursor--
		}

	case tea.KeyRight, tea.KeyCtrlF:
		if m.cursor < len(m.runes) {
			m.cursor++
		}

	case tea.KeyBackspace, tea.KeyCtrlH:
		// Backspace sends DEL, Ctrl+H sends BS; treat both as backspace.
		// Alt+Backspace (the word-delete combo) is dropped rather than
		// falling through as a plain backspace.
		if key.Alt {
			return m, nil
		}
		if m.cursor == 0 {
			return m, nil
		}
		m.runes = append(m.runes[:m.cursor-1], m.runes[m.cursor:]...)
		m.cursor--
		m.submitErr = nil

	case tea.KeyRunes, tea.KeySpace:
		// Alt+<rune> (e.g. Alt+f, Alt+b) are word-nav combos we do not
		// support; drop them rather than inserting the rune literally.
		if key.Alt {
			return m, nil
		}
		typed := key.Runes
		if key.Type == tea.KeySpace {
			typed = []rune{' '}
		}
		tail := append([]rune{}, m.runes[m.cursor:]...)
		m.runes = append(m.runes[:m.cursor], typed...)
		m.runes = append(m.runes, tail...)
		m.cursor += len(typed)
		m.submitErr = nil

	default:
		// All other key types are intentionally inert (Home/End,
		// Ctrl+W/Ctrl+U, Ctrl+P/N, function keys, etc.).
	}

	return m, nil
}

func (m *promptModel) View() string {
	// HideEntered: empty final frame so the masked value isn't left in
	// scrollback after the user presses Enter.
	if m.submitted && m.hideEntered {
		return ""
	}

	display := m.runes
	if m.mask != 0 {
		display = make([]rune, len(m.runes))
		for i := range display {
			display[i] = m.mask
		}
	}

	// Post-submit frame: faint label, faint colon, then the entered value
	// plain. No cursor block.
	//
	// The trailing "\n" is load-bearing. On tea.Quit, bubbletea's renderer
	// flushes one last frame (so this View runs with submitted=true), then
	// stop() runs `EraseEntireLine` + "\r" to park the cursor cleanly for
	// whatever output follows. EraseEntireLine wipes the row the cursor is
	// on — so we end the frame with "\n" to advance the cursor onto an
	// empty sacrificial row, leaving the rendered text intact above. Pre-
	// submit frames must NOT trail "\n" or every keystroke would consume
	// an extra terminal row and risk scrolling at the screen bottom.
	if m.submitted {
		return ansiFaint + m.label + ":" + ansiReset + " " + string(display) + "\n"
	}

	prefix := m.glyph() + " " + ansiBold + m.label + ansiReset + ansiBold + ":" + ansiReset + " "

	var line string
	if m.cursor >= len(display) {
		line = prefix + string(display) + cursorBlock
	} else {
		// Cursor block visually replaces the rune at the cursor; the hidden
		// rune is still in m.runes and reappears once the cursor moves.
		line = prefix + string(display[:m.cursor]) + cursorBlock + string(display[m.cursor+1:])
	}

	if m.submitErr != nil {
		// Validation error line captured at the failed Enter; cleared on
		// the next edit.
		line += "\n" + ansiRed + ">> " + m.submitErr.Error() + ansiReset
	}
	return line
}

func (c *cmdIO) runPromptModel(m *promptModel) (string, error) {
	final, err := c.runTUI(m)
	if err != nil {
		return "", err
	}
	pm := final.(*promptModel)
	switch {
	case pm.cancelled:
		return "", errCtrlC
	case pm.deleted:
		return "", io.EOF
	}
	return strings.TrimRight(pm.value(), "\r\n"), nil
}
