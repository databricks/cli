package aircmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

// asciiPalette returns a palette whose styles emit no escape codes, so render
// output is plain text and assertions stay readable.
func asciiPalette(t *testing.T) palette {
	r, _ := cmdio.NewRenderer(cmdio.MockDiscard(t.Context()), io.Discard)
	return newPalette(r)
}

// A local HTTP server exercises the SDK export path without a live workspace.
func configTestClient(t *testing.T, path, content string) *databricks.WorkspaceClient {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/2.0/workspace/export" {
			assert.Equal(t, path, r.URL.Query().Get("path"))
			assert.Equal(t, "true", r.URL.Query().Get("direct_download"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			body, err := io.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Empty(t, body)
			_, _ = io.WriteString(w, content)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	}))
	t.Cleanup(srv.Close)
	return newTestWorkspaceClient(t, srv.URL)
}

func TestConfigYAML(t *testing.T) {
	ctx := t.Context()

	t.Run("inline parameters include the command as a block literal", func(t *testing.T) {
		task := &jobs.GenAiComputeTask{
			YamlParameters: "experiment_name: my-exp\ncompute:\n  accelerator_type: a10\n  num_accelerators: 1\ncommand: \"for i in $(seq 1 3); do echo $i; done\\n\"\n",
		}
		got, _ := configYAML(ctx, mocks.NewMockWorkspaceClient(t).WorkspaceClient, task)
		assert.Contains(t, got, "experiment_name: my-exp")
		assert.Contains(t, got, "accelerator_type: a10")
		assert.Contains(t, got, "command: |")
		assert.Contains(t, got, "  for i in $(seq 1 3); do echo $i; done")
	})

	t.Run("downloads the parameters file when there are no inline parameters", func(t *testing.T) {
		w := configTestClient(t, "/Workspace/cfg.yaml", "experiment_name: from-file\n")
		task := &jobs.GenAiComputeTask{YamlParametersFilePath: "/Workspace/cfg.yaml"}
		got, raw := configYAML(ctx, w, task)
		assert.Equal(t, "experiment_name: from-file", got)
		assert.Equal(t, "experiment_name: from-file\n", string(raw))
	})

	t.Run("falls back to a synthesized config", func(t *testing.T) {
		task := &jobs.GenAiComputeTask{
			MlflowExperimentName: "/Users/me@example.com/exp",
			Compute:              &jobs.ComputeConfig{GpuType: "a10", NumGpus: 1},
		}
		got, _ := configYAML(ctx, mocks.NewMockWorkspaceClient(t).WorkspaceClient, task)
		assert.Contains(t, got, "experiment_name: exp")
		assert.NotContains(t, got, "command")
	})
}

func TestDownloadConfigSizeLimit(t *testing.T) {
	for _, tc := range []struct {
		name    string
		size    int
		wantNil bool
	}{
		{"at limit", maxConfigYAMLBytes, false},
		{"above limit", maxConfigYAMLBytes + 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := configTestClient(t, "/Workspace/cfg.yaml", strings.Repeat("a", tc.size))
			got := downloadConfig(withQuietEnrichment(t.Context()), w, "/Workspace/cfg.yaml")
			if tc.wantNil {
				assert.Nil(t, got)
			} else {
				assert.Len(t, got, tc.size)
			}
		})
	}
}

type configCountingReader struct {
	reader    io.Reader
	bytesRead int
}

func (r *configCountingReader) Read(p []byte) (int, error) {
	n, err := r.reader.Read(p)
	r.bytesRead += n
	return n, err
}

func TestDownloadConfigLimitsSDKResponseReads(t *testing.T) {
	for _, contentType := range []string{"application/json", "application/octet-stream"} {
		t.Run(contentType, func(t *testing.T) {
			body := &configCountingReader{reader: strings.NewReader(strings.Repeat("x", 4*maxConfigYAMLBytes))}
			// The transport simulates a large workspace export without a live service.
			transport := validationRoundTripFunc(func(req *http.Request) (*http.Response, error) {
				if req.URL.Path != "/api/2.0/workspace/export" {
					return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(`{}`)), Request: req}, nil
				}
				return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(body), Request: req}, nil
			})
			w := validationTestWorkspaceClientWithTransport(t, transport)
			got := downloadConfig(withQuietEnrichment(t.Context()), w, "/Workspace/cfg.yaml")
			assert.Nil(t, got)
			assert.LessOrEqual(t, body.bytesRead, maxConfigYAMLBytes+1)
		})
	}
}

func TestResolveConfigYAML(t *testing.T) {
	ctx := t.Context()

	t.Run("downloads the training config path for new runs", func(t *testing.T) {
		w := configTestClient(t, "/Workspace/run/training_config.yaml", "experiment_name: from-air\n")
		run := &jobs.Run{Tasks: []jobs.RunTask{{}}}
		data := &getData{TrainingConfigPath: "/Workspace/run/training_config.yaml"}
		got, raw := resolveConfigYAML(ctx, w, run, data)
		assert.Equal(t, "experiment_name: from-air", got)
		assert.Equal(t, "experiment_name: from-air\n", string(raw))
	})

	t.Run("falls back to the legacy task when no path is set", func(t *testing.T) {
		run := &jobs.Run{Tasks: []jobs.RunTask{{GenAiComputeTask: &jobs.GenAiComputeTask{
			MlflowExperimentName: "/Users/me@example.com/exp",
		}}}}
		got, _ := resolveConfigYAML(ctx, mocks.NewMockWorkspaceClient(t).WorkspaceClient, run, &getData{})
		assert.Contains(t, got, "experiment_name: exp")
	})

	t.Run("empty when neither source is present", func(t *testing.T) {
		run := &jobs.Run{Tasks: []jobs.RunTask{{}}}
		got, raw := resolveConfigYAML(ctx, mocks.NewMockWorkspaceClient(t).WorkspaceClient, run, &getData{})
		assert.Empty(t, got)
		assert.Nil(t, raw)
	})
}

func TestSynthConfigYAML(t *testing.T) {
	task := &jobs.GenAiComputeTask{
		MlflowExperimentName: "/Users/me@example.com/stream-latency-test",
		Compute:              &jobs.ComputeConfig{GpuType: "a10", NumGpus: 1},
	}
	// The accelerator_type uses the raw GPU type; the command is omitted because
	// it lives only in the run parameters.
	want := "experiment_name: stream-latency-test\n" +
		"compute:\n" +
		"  accelerator_type: a10\n" +
		"  num_accelerators: 1"
	assert.Equal(t, want, synthConfigYAML(task))
	assert.Empty(t, synthConfigYAML(&jobs.GenAiComputeTask{}))
}

func TestColorizeConfigLine(t *testing.T) {
	p := asciiPalette(t)
	// Under the Ascii profile colorization adds no escapes, so each line is
	// preserved verbatim (indentation included) regardless of its role.
	for _, line := range []string{
		"experiment_name: stream-latency-test",
		"compute:",
		"  accelerator_type: a10",
		"  num_accelerators: 1",
		"command: |",
		`  for i in $(seq 1 10); do echo "step $i"; done`,
	} {
		assert.Equal(t, line, colorizeConfigLine(p, line))
	}
}

func TestIsConfigKey(t *testing.T) {
	assert.True(t, isConfigKey("experiment_name"))
	assert.True(t, isConfigKey("num_accelerators"))
	assert.False(t, isConfigKey(""))
	assert.False(t, isConfigKey("for i in $(seq 1 10); do echo "))
	assert.False(t, isConfigKey("Command"))
}

func TestRenderBox(t *testing.T) {
	p := asciiPalette(t)
	out := renderBox(p, configBoxTitle, "experiment_name: stream-latency-test\ncompute:")
	lines := strings.Split(out, "\n")

	// Title sits in the top border; corners are rounded; every row is the same width.
	assert.Contains(t, lines[0], "╭─ "+configBoxTitle+" ")
	assert.True(t, strings.HasSuffix(lines[0], "╮"))
	assert.True(t, strings.HasPrefix(lines[len(lines)-1], "╰"))
	assert.Contains(t, out, "│  experiment_name: stream-latency-test")

	width := lipgloss.Width(lines[0])
	for _, l := range lines {
		assert.Equal(t, width, lipgloss.Width(l))
	}
}

func TestRenderFields(t *testing.T) {
	p := asciiPalette(t)
	out := renderFields(p, false, runView{
		runID:        "836121283738861",
		dashboardURL: "https://h.test/jobs/runs/836121283738861",
		status:       "SUCCESS",
		submitted:    "2026-06-03 04:17 UTC",
		retries:      0,
		maxRetries:   "3",
		duration:     "1m 13s",
		experiment:   "stream-latency-test",
		mlflowLabel:  "stream-latency-test",
		mlflowURL:    "https://h.test/ml/experiments/E1/runs/R1",
		user:         "user@example.com",
		accelerators: "1x A10",
		environment:  "4",
	})

	// Labels are padded to the longest ("Accelerators"), so values align.
	assert.Contains(t, out, "Run ID        ")
	assert.Contains(t, out, "Accelerators  1x A10")
	// Max retries and environment show alongside the other fields.
	assert.Contains(t, out, "Max Retries   3")
	assert.Contains(t, out, "Environment   4")
	// The status carries its dot prefix.
	assert.Contains(t, out, "● SUCCESS")
	// Off a terminal, links render as the bare label (URLs live in JSON output).
	assert.Contains(t, out, "Run ID        836121283738861")
	assert.NotContains(t, out, "https://h.test")
	// The field list is a tight block: no blank lines.
	assert.NotContains(t, out, "\n\n")
}

func TestLink(t *testing.T) {
	p := asciiPalette(t)
	// Color off: the bare label, no URL.
	assert.Equal(t, "label", link(false, p.blue, "label", "https://h.test"))
	assert.Equal(t, "label", link(false, p.blue, "label", ""))
	// With color on, the label is wrapped in an OSC 8 hyperlink to the url.
	assert.Contains(t, link(true, p.blue, "label", "https://h.test"), termenv.Hyperlink("https://h.test", "label"))
}

func TestStatusStyleSelectors(t *testing.T) {
	assert.True(t, isSuccessStatus("SUCCESS"))
	assert.False(t, isSuccessStatus("RUNNING"))
	assert.True(t, isFailedStatus("FAILED"))
	assert.True(t, isFailedStatus("TIMEDOUT"))
	assert.False(t, isFailedStatus("RUNNING"))
}
