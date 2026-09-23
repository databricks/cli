package pipelines

import (
	"encoding/json"
	"regexp"
	"strings"
)

// ansiEscapeText matches terminal color codes that the pipeline test emitter
// captures as *literal* text (the six-character sequence \u001B[..m, not real
// ESC control bytes) inside failure messages. They are stripped for display;
// removing color codes changes no semantic content of the pytest output.
var ansiEscapeText = regexp.MustCompile(`\\u001[bB]\[[0-9;]*m`)

// rawTestResult mirrors the nested PipelineTestCaseProgress.result object of the
// production /events (event_record) wire shape.
type rawTestResult struct {
	Outcome    string `json:"outcome"`
	DurationMs *int64 `json:"duration_ms"`
	SkipReason string `json:"skip_reason"`
	Failure    *struct {
		Exceptions []struct {
			Message   string `json:"message"`
			Traceback string `json:"traceback"`
		} `json:"exceptions"`
	} `json:"failure"`
}

// UnmarshalJSON decodes the real production test_case_progress proto JSON (as
// served by /events and persisted in the CP event_record table) into the flat
// internal testCaseProgress the renderers consume. The wire shape carries
// enum-prefixed status/outcome values (PIPELINE_TEST_CASE_STATUS_*,
// PIPELINE_TEST_CASE_OUTCOME_*), a nested source, and a nested result object; a
// bare-string result and flat path/line are also tolerated so older simplified
// fixtures still decode.
func (c *testCaseProgress) UnmarshalJSON(data []byte) error {
	var raw struct {
		NodeID string `json:"node_id"`
		Source *struct {
			WorkspacePath string `json:"workspace_path"`
			Line          *int   `json:"line"`
		} `json:"source"`
		Status     string          `json:"status"`
		Result     json.RawMessage `json:"result"`
		Path       string          `json:"path"`
		Line       *int            `json:"line"`
		DurationMs *int64          `json:"duration_ms"`
		Message    string          `json:"message"`
		Traceback  string          `json:"traceback"`
		Truncated  bool            `json:"truncated"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	c.NodeID = raw.NodeID
	c.Status = strings.TrimPrefix(raw.Status, "PIPELINE_TEST_CASE_STATUS_")
	c.Truncated = raw.Truncated
	c.DurationMs = raw.DurationMs
	c.Message = cleanTestText(raw.Message)
	c.Traceback = cleanTestText(raw.Traceback)

	// Identity (path/line): prefer the nested proto source, fall back to flat.
	c.Path = raw.Path
	c.Line = raw.Line
	if raw.Source != nil {
		if c.Path == "" {
			c.Path = raw.Source.WorkspacePath
		}
		if c.Line == nil {
			c.Line = raw.Source.Line
		}
	}

	if len(raw.Result) == 0 || string(raw.Result) == "null" {
		return nil
	}

	// Simplified shape: result is a bare outcome string.
	var flat string
	if err := json.Unmarshal(raw.Result, &flat); err == nil {
		c.Result = strings.TrimPrefix(flat, "PIPELINE_TEST_CASE_OUTCOME_")
		return nil
	}

	// Production shape: result is a nested object.
	var ro rawTestResult
	if err := json.Unmarshal(raw.Result, &ro); err != nil {
		return err
	}
	c.Result = strings.TrimPrefix(ro.Outcome, "PIPELINE_TEST_CASE_OUTCOME_")
	if ro.DurationMs != nil {
		c.DurationMs = ro.DurationMs
	}
	switch {
	case ro.SkipReason != "":
		if c.Message == "" {
			c.Message = cleanTestText(ro.SkipReason)
		}
	case ro.Failure != nil && len(ro.Failure.Exceptions) > 0:
		full := cleanTestText(ro.Failure.Exceptions[0].Message)
		tb := cleanTestText(ro.Failure.Exceptions[0].Traceback)
		if tb == "" {
			tb = full
		}
		if c.Traceback == "" {
			c.Traceback = tb
		}
		if c.Message == "" {
			c.Message = conciseErrorLine(full)
		}
	}
	return nil
}

// cleanTestText normalizes captured pytest failure text: it strips literal ANSI
// color codes and un-escapes the newline / tab / quote sequences the emitter
// captured as literal two-character text (\n, \t, \"). This changes only
// presentation, not the semantic content of the failure.
func cleanTestText(s string) string {
	if s == "" {
		return s
	}
	s = ansiEscapeText.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\t", "\t")
	s = strings.ReplaceAll(s, "\\\"", "\"")
	return s
}

// conciseErrorLine returns the pytest "E   <Error>: <message>" summary line
// from a full failure text, or the last non-empty line as a fallback.
func conciseErrorLine(text string) string {
	var last string
	for ln := range strings.SplitSeq(text, "\n") {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		last = t
		if strings.HasPrefix(t, "E ") || strings.HasPrefix(t, "E\t") {
			return strings.TrimSpace(t[1:])
		}
	}
	return last
}
