package executor

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	sqlapi "github.com/databricks/databricks-sdk-go/service/sql"
)

type fakeRoundTripper struct {
	req  *http.Request
	resp *http.Response
	err  error
}

func (f *fakeRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	f.req = r
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

func TestExternalLinkDownloaderUsesProvidedHeaders(t *testing.T) {
	fake := &fakeRoundTripper{
		resp: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"result":{"data_array":[["value"]]}}`)),
		},
	}
	d := &externalLinkDownloader{client: fake}
	rows, err := d.download(context.Background(), sqlapi.ExternalLink{
		ExternalLink: "https://example.com/chunk",
		HttpHeaders: map[string]string{
			"Authorization": "Bearer signed",
		},
		ChunkIndex: 3,
	})
	require.NoError(t, err)
	require.Equal(t, [][]string{{"value"}}, rows)
	require.Equal(t, "Bearer signed", fake.req.Header.Get("Authorization"))
}

func TestExternalLinkDownloaderSupportsTopLevelArray(t *testing.T) {
	fake := &fakeRoundTripper{
		resp: &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`[["value"]]`)),
		},
	}
	d := &externalLinkDownloader{client: fake}
	rows, err := d.download(context.Background(), sqlapi.ExternalLink{
		ExternalLink: "https://example.com/chunk",
		ChunkIndex:   4,
	})
	require.NoError(t, err)
	require.Equal(t, [][]string{{"value"}}, rows)
}

func TestTableSinkPrintsDividerUnderHeader(t *testing.T) {
	var buf bytes.Buffer
	sink := newTableSink(&buf)
	require.NoError(t, sink.Begin("stmt", []string{"foo", "bar"}))
	require.NoError(t, sink.Row([]string{"foo", "bar"}, []string{"1", "2"}))
	require.NoError(t, sink.End("stmt", nil))
	out := buf.String()
	require.Contains(t, out, "foo")
	require.Contains(t, out, "bar")
	require.Contains(t, out, "───")
}

func TestRenderStatementUsesInlineChunkWithoutFetching(t *testing.T) {
	getter := &stubChunkGetter{chunks: map[int]*sqlapi.ResultData{}}
	sink := &recordingSink{}
	resp := &sqlapi.StatementResponse{
		StatementId: "stmt-inline",
		Result: &sqlapi.ResultData{
			ChunkIndex: 5,
			DataArray:  [][]string{{"inline"}},
		},
	}

	require.NoError(t, renderStatement(context.Background(), getter, resp, sink, nil))
	require.Empty(t, getter.calls)
	require.Equal(t, [][]string{{"inline"}}, sink.rows)
}

func TestRenderStatementStreamsManifestInOrder(t *testing.T) {
	getter := &stubChunkGetter{
		chunks: map[int]*sqlapi.ResultData{
			8: {
				ChunkIndex: 8,
				DataArray:  [][]string{{"fetched"}},
			},
		},
	}
	sink := &recordingSink{}
	resp := &sqlapi.StatementResponse{
		StatementId: "stmt-manifest",
		Manifest: &sqlapi.ResultManifest{
			Chunks: []sqlapi.BaseChunkInfo{
				{ChunkIndex: 7},
				{ChunkIndex: 8},
			},
		},
		Result: &sqlapi.ResultData{
			ChunkIndex: 7,
			DataArray:  [][]string{{"inline"}},
		},
	}

	require.NoError(t, renderStatement(context.Background(), getter, resp, sink, nil))
	require.Equal(t, []int{8}, getter.calls)
	require.Equal(t, [][]string{{"inline"}, {"fetched"}}, sink.rows)
}

func TestEnsureSupportedFormatAllowsJSON(t *testing.T) {
	resp := &sqlapi.StatementResponse{
		Manifest: &sqlapi.ResultManifest{Format: sqlapi.FormatJsonArray},
	}
	require.NoError(t, ensureSupportedFormat(resp))
}

func TestEnsureSupportedFormatRejectsArrow(t *testing.T) {
	resp := &sqlapi.StatementResponse{
		Manifest: &sqlapi.ResultManifest{Format: sqlapi.FormatArrowStream},
	}
	err := ensureSupportedFormat(resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "arrow output is not supported")
}

func TestEnsureSupportedFormatRejectsUnknown(t *testing.T) {
	resp := &sqlapi.StatementResponse{
		Manifest: &sqlapi.ResultManifest{Format: "PARQUET"},
	}
	err := ensureSupportedFormat(resp)
	require.Error(t, err)
	require.Contains(t, err.Error(), "result format PARQUET")
}

func TestExpandChunkReturnsDownloaderError(t *testing.T) {
	chunk := &sqlapi.ResultData{
		ExternalLinks: []sqlapi.ExternalLink{{ExternalLink: "https://example.com", ChunkIndex: 1}},
	}
	_, err := expandChunk(context.Background(), chunk, &externalLinkDownloader{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "workspace configuration is not available")
}

type stubChunkGetter struct {
	calls  []int
	chunks map[int]*sqlapi.ResultData
}

func (s *stubChunkGetter) GetStatementResultChunkN(ctx context.Context, req sqlapi.GetStatementResultChunkNRequest) (*sqlapi.ResultData, error) {
	s.calls = append(s.calls, req.ChunkIndex)
	chunk, ok := s.chunks[req.ChunkIndex]
	if !ok {
		panic("unexpected chunk index")
	}
	return chunk, nil
}

type recordingSink struct {
	rows [][]string
}

func (r *recordingSink) Begin(string, []string) error {
	return nil
}

func (r *recordingSink) Row(_, row []string) error {
	r.rows = append(r.rows, append([]string(nil), row...))
	return nil
}

func (r *recordingSink) End(string, []string) error {
	return nil
}
