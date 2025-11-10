package executor

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/databricks/cli/libs/safety/sqlsafe"
	databricks "github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/httpclient"
	sqlapi "github.com/databricks/databricks-sdk-go/service/sql"
)

// Format enumerates the supported output formats.
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatCSV   Format = "csv"
)

// Options configures statement execution.
type Options struct {
	Context        context.Context
	Client         *databricks.WorkspaceClient
	Statements     []sqlsafe.Statement
	WarehouseID    string
	WaitTimeout    string
	Format         Format
	Output         io.Writer
	Stderr         io.Writer
	SpinnerFactory func(context.Context) chan string
	LogString      func(context.Context, string)
}

// Run executes statements according to the provided options.
func Run(opts Options) error {
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}
	if opts.Client == nil {
		return errors.New("workspace client is required")
	}
	stdout := opts.Output
	if stdout == nil {
		stdout = io.Discard
	}
	stderr := opts.Stderr
	if stderr == nil {
		stderr = io.Discard
	}
	logFn := opts.LogString
	if logFn == nil {
		logFn = func(context.Context, string) {}
	}
	spinnerFactory := opts.SpinnerFactory
	if spinnerFactory == nil {
		spinnerFactory = func(context.Context) chan string { return nil }
	}

	downloader := newExternalLinkDownloader(opts.Client.Config)

	stopSpinner := func(ch chan string) {
		if ch != nil {
			close(ch)
		}
	}

	for idx, stmt := range opts.Statements {
		var spinner chan string
		if opts.WaitTimeout != "0s" {
			spinner = spinnerFactory(ctx)
		}
		text := stmt.OriginalText
		if strings.TrimSpace(text) == "" {
			text = stmt.Text
		}
		if strings.TrimSpace(text) == "" {
			continue
		}

		if spinner != nil {
			spinner <- fmt.Sprintf("Submitting statement %d", idx+1)
		}

		resp, err := opts.Client.StatementExecution.ExecuteStatement(ctx, sqlapi.ExecuteStatementRequest{
			WarehouseId:   opts.WarehouseID,
			Statement:     text,
			WaitTimeout:   opts.WaitTimeout,
			OnWaitTimeout: sqlapi.ExecuteStatementRequestOnWaitTimeoutContinue,
			Disposition:   sqlapi.DispositionExternalLinks,
		})
		if err != nil {
			stopSpinner(spinner)
			return err
		}

		if opts.WaitTimeout == "0s" {
			logFn(ctx, fmt.Sprintf("Statement %d submitted (id: %s). Poll via GET /api/2.0/sql/statements/%s", idx+1, resp.StatementId, resp.StatementId))
			stopSpinner(spinner)
			continue
		}

		finalResp, err := waitForCompletion(ctx, opts.Client, resp, spinner)
		stopSpinner(spinner)
		if err != nil {
			return err
		}

		if err := ensureSupportedFormat(finalResp); err != nil {
			return err
		}

		if idx > 0 {
			switch opts.Format {
			case FormatTable, FormatCSV:
				fmt.Fprintln(stdout)
			case FormatJSON:
				// NDJSON blocks do not need separators between statements.
			}
		}
		if opts.Format == FormatTable && len(opts.Statements) > 1 {
			fmt.Fprintf(stdout, "Statement %d (id: %s)\n", idx+1, finalResp.StatementId)
		}

		sink, err := newRowSink(opts.Format, stdout)
		if err != nil {
			return err
		}

		if err := renderStatement(ctx, opts.Client.StatementExecution, finalResp, sink, downloader); err != nil {
			return err
		}

		if finalResp.Manifest != nil && finalResp.Manifest.Truncated {
			fmt.Fprintf(stderr, "Results truncated for statement %s\n", finalResp.StatementId)
		}
	}

	return nil
}

func waitForCompletion(ctx context.Context, client *databricks.WorkspaceClient, resp *sqlapi.StatementResponse, spinner chan string) (*sqlapi.StatementResponse, error) {
	current := resp
	state := getState(current)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for state == sqlapi.StatementStatePending || state == sqlapi.StatementStateRunning {
		if spinner != nil {
			spinner <- fmt.Sprintf("Statement %s is %s", current.StatementId, strings.ToLower(state.String()))
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}

		next, err := client.StatementExecution.GetStatementByStatementId(ctx, current.StatementId)
		if err != nil {
			return nil, err
		}
		current = next
		state = getState(current)
	}
	return current, ensureSucceeded(current)
}

func ensureSucceeded(resp *sqlapi.StatementResponse) error {
	if resp == nil || resp.Status == nil {
		return errors.New("statement failed")
	}
	switch resp.Status.State {
	case sqlapi.StatementStateSucceeded, sqlapi.StatementStateClosed:
		return nil
	case sqlapi.StatementStateCanceled:
		return fmt.Errorf("statement %s was canceled", resp.StatementId)
	case sqlapi.StatementStateFailed:
		if resp.Status.Error != nil {
			return fmt.Errorf("statement %s failed: %s", resp.StatementId, resp.Status.Error.Message)
		}
		return fmt.Errorf("statement %s failed", resp.StatementId)
	default:
		return fmt.Errorf("statement %s ended in state %s", resp.StatementId, resp.Status.State)
	}
}

func getState(resp *sqlapi.StatementResponse) sqlapi.StatementState {
	if resp == nil || resp.Status == nil {
		return ""
	}
	return resp.Status.State
}

func ensureSupportedFormat(resp *sqlapi.StatementResponse) error {
	if resp == nil || resp.Manifest == nil {
		return nil
	}
	switch resp.Manifest.Format {
	case "", sqlapi.FormatJsonArray:
		return nil
	case sqlapi.FormatArrowStream:
		return errors.New("arrow output is not supported yet; rerun with format=JSON_ARRAY or --format json")
	default:
		return fmt.Errorf("result format %s is not supported; rerun with --format json", resp.Manifest.Format)
	}
}

type statementResultChunkGetter interface {
	GetStatementResultChunkN(ctx context.Context, request sqlapi.GetStatementResultChunkNRequest) (*sqlapi.ResultData, error)
}

func renderStatement(ctx context.Context, chunkGetter statementResultChunkGetter, resp *sqlapi.StatementResponse, sink rowSink, downloader *externalLinkDownloader) error {
	if resp == nil {
		return nil
	}

	columns := columnNamesFromManifest(resp.Manifest)
	if err := sink.Begin(resp.StatementId, columns); err != nil {
		return err
	}

	inlineChunk := resp.Result
	manifest := resp.Manifest

	if manifest == nil || len(manifest.Chunks) == 0 {
		if err := streamChunk(ctx, inlineChunk, manifest, sink, downloader, &columns); err != nil {
			return err
		}
		return sink.End(resp.StatementId, columns)
	}

	for _, chunk := range manifest.Chunks {
		if inlineChunk != nil && chunk.ChunkIndex == inlineChunk.ChunkIndex {
			if err := streamChunk(ctx, inlineChunk, manifest, sink, downloader, &columns); err != nil {
				return err
			}
			inlineChunk = nil
			continue
		}

		if chunkGetter == nil {
			return errors.New("statement chunk getter not initialized")
		}
		respChunk, err := chunkGetter.GetStatementResultChunkN(ctx, sqlapi.GetStatementResultChunkNRequest{
			StatementId: resp.StatementId,
			ChunkIndex:  chunk.ChunkIndex,
		})
		if err != nil {
			return err
		}
		if err := streamChunk(ctx, respChunk, manifest, sink, downloader, &columns); err != nil {
			return err
		}
	}

	if inlineChunk != nil {
		if err := streamChunk(ctx, inlineChunk, manifest, sink, downloader, &columns); err != nil {
			return err
		}
	}

	return sink.End(resp.StatementId, columns)
}

func streamChunk(ctx context.Context, chunk *sqlapi.ResultData, manifest *sqlapi.ResultManifest, sink rowSink, downloader *externalLinkDownloader, columns *[]string) error {
	chunks, err := expandChunk(ctx, chunk, downloader)
	if err != nil {
		return err
	}
	for _, c := range chunks {
		if len(*columns) == 0 {
			*columns = inferColumns(manifest, c)
		}
		if len(*columns) == 0 {
			continue
		}
		for _, row := range c.DataArray {
			if err := sink.Row(*columns, row); err != nil {
				return err
			}
		}
	}
	return nil
}

func expandChunk(ctx context.Context, chunk *sqlapi.ResultData, downloader *externalLinkDownloader) ([]*sqlapi.ResultData, error) {
	if chunk == nil {
		return nil, nil
	}
	if len(chunk.ExternalLinks) == 0 {
		return []*sqlapi.ResultData{chunk}, nil
	}
	if downloader == nil {
		return nil, errors.New("external link downloader not initialized")
	}

	results := make([]*sqlapi.ResultData, 0, len(chunk.ExternalLinks))
	for _, link := range chunk.ExternalLinks {
		rows, err := downloader.download(ctx, link)
		if err != nil {
			return nil, err
		}
		copyChunk := *chunk
		copyChunk.ExternalLinks = nil
		copyChunk.DataArray = rows
		copyChunk.ChunkIndex = link.ChunkIndex
		copyChunk.RowCount = link.RowCount
		copyChunk.RowOffset = link.RowOffset
		results = append(results, &copyChunk)
	}
	return results, nil
}

func columnNamesFromManifest(manifest *sqlapi.ResultManifest) []string {
	if manifest == nil || manifest.Schema == nil {
		return nil
	}
	cols := make([]string, len(manifest.Schema.Columns))
	for i, col := range manifest.Schema.Columns {
		name := col.Name
		if name == "" {
			name = fmt.Sprintf("col_%d", i+1)
		}
		cols[i] = name
	}
	return cols
}

func inferColumns(manifest *sqlapi.ResultManifest, chunk *sqlapi.ResultData) []string {
	if names := columnNamesFromManifest(manifest); len(names) > 0 {
		return names
	}
	if chunk != nil && len(chunk.DataArray) > 0 {
		rowLen := len(chunk.DataArray[0])
		cols := make([]string, rowLen)
		for i := range cols {
			cols[i] = fmt.Sprintf("col_%d", i+1)
		}
		return cols
	}
	return nil
}

type rowSink interface {
	Begin(statementID string, columns []string) error
	Row(columns, row []string) error
	End(statementID string, columns []string) error
}

func newRowSink(format Format, out io.Writer) (rowSink, error) {
	switch format {
	case FormatTable:
		return newTableSink(out), nil
	case FormatCSV:
		return newCSVSink(out), nil
	case FormatJSON:
		return newNDJSONSink(out), nil
	default:
		return nil, fmt.Errorf("unsupported format %q", format)
	}
}

type tableSink struct {
	out        io.Writer
	writer     *tabwriter.Writer
	columns    []string
	rowBuf     []string
	headerSet  bool
	statement  string
	columnsSet bool
}

func newTableSink(out io.Writer) *tableSink {
	return &tableSink{out: out}
}

func (s *tableSink) Begin(statementID string, columns []string) error {
	s.statement = statementID
	s.columns = nil
	s.rowBuf = nil
	s.headerSet = false
	s.writer = nil
	s.columnsSet = false
	if len(columns) > 0 {
		s.columns = append([]string(nil), columns...)
		s.columnsSet = true
		s.ensureWriter()
		s.ensureHeader()
	}
	return nil
}

func (s *tableSink) Row(columns, row []string) error {
	if len(columns) == 0 {
		return nil
	}
	if !s.columnsSet {
		s.columns = append([]string(nil), columns...)
		s.columnsSet = true
	}
	s.ensureWriter()
	s.ensureHeader()
	s.writeRow(row)
	return nil
}

func (s *tableSink) End(statementID string, _ []string) error {
	if s.writer != nil {
		return s.writer.Flush()
	}
	fmt.Fprintf(s.out, "Statement %s returned no results\n", statementID)
	return nil
}

func (s *tableSink) ensureWriter() {
	if s.writer == nil {
		s.writer = tabwriter.NewWriter(s.out, 0, 8, 2, ' ', 0)
	}
}

func (s *tableSink) ensureHeader() {
	if s.writer == nil || s.headerSet || len(s.columns) == 0 {
		return
	}
	fmt.Fprintln(s.writer, strings.Join(s.columns, "\t"))
	dividers := make([]string, len(s.columns))
	for i, col := range s.columns {
		width := len(col)
		if width < 3 {
			width = 3
		}
		dividers[i] = strings.Repeat("─", width)
	}
	fmt.Fprintln(s.writer, strings.Join(dividers, "\t"))
	s.headerSet = true
}

func (s *tableSink) writeRow(row []string) {
	if len(s.rowBuf) != len(s.columns) {
		s.rowBuf = make([]string, len(s.columns))
	}
	copy(s.rowBuf, row)
	for i := len(row); i < len(s.rowBuf); i++ {
		s.rowBuf[i] = ""
	}
	for i, val := range s.rowBuf {
		if i > 0 {
			_, _ = fmt.Fprint(s.writer, "\t")
		}
		_, _ = fmt.Fprint(s.writer, val)
	}
	_, _ = fmt.Fprintln(s.writer)
}

type csvSink struct {
	writer    *csv.Writer
	columns   []string
	rowBuf    []string
	headerSet bool
}

func newCSVSink(out io.Writer) *csvSink {
	return &csvSink{writer: csv.NewWriter(out)}
}

func (s *csvSink) Begin(_ string, columns []string) error {
	s.columns = nil
	s.rowBuf = nil
	s.headerSet = false
	if len(columns) > 0 {
		s.columns = append([]string(nil), columns...)
	}
	return nil
}

func (s *csvSink) Row(columns, row []string) error {
	if len(columns) == 0 {
		return nil
	}
	if len(s.columns) == 0 {
		s.columns = append([]string(nil), columns...)
	}
	if !s.headerSet {
		if err := s.writer.Write(s.columns); err != nil {
			return err
		}
		s.headerSet = true
	}
	return s.writer.Write(s.projectRow(row))
}

func (s *csvSink) End(string, []string) error {
	s.writer.Flush()
	return s.writer.Error()
}

func (s *csvSink) projectRow(row []string) []string {
	if len(s.rowBuf) != len(s.columns) {
		s.rowBuf = make([]string, len(s.columns))
	}
	copy(s.rowBuf, row)
	for i := len(row); i < len(s.rowBuf); i++ {
		s.rowBuf[i] = ""
	}
	return s.rowBuf
}

type ndjsonSink struct {
	enc     *json.Encoder
	columns []string
	record  map[string]string
}

func newNDJSONSink(out io.Writer) *ndjsonSink {
	enc := json.NewEncoder(out)
	enc.SetEscapeHTML(false)
	return &ndjsonSink{enc: enc}
}

func (s *ndjsonSink) Begin(_ string, columns []string) error {
	s.columns = nil
	s.record = nil
	if len(columns) > 0 {
		s.columns = append([]string(nil), columns...)
	}
	return nil
}

func (s *ndjsonSink) Row(columns, row []string) error {
	if len(columns) == 0 {
		return nil
	}
	if len(s.columns) == 0 {
		s.columns = append([]string(nil), columns...)
	}
	rec := s.resetRecord()
	for i, name := range s.columns {
		val := ""
		if i < len(row) {
			val = row[i]
		}
		rec[name] = val
	}
	return s.enc.Encode(rec)
}

func (s *ndjsonSink) End(string, []string) error {
	return nil
}

func (s *ndjsonSink) resetRecord() map[string]string {
	if s.record == nil {
		s.record = make(map[string]string)
	}
	for k := range s.record {
		delete(s.record, k)
	}
	return s.record
}

type externalLinkPayload struct {
	Result struct {
		DataArray [][]string `json:"data_array"`
	} `json:"result"`
	DataArray [][]string `json:"data_array"`
}

type httpRoundTripper interface {
	RoundTrip(*http.Request) (*http.Response, error)
}

type externalLinkDownloader struct {
	cfg    *config.Config
	once   sync.Once
	client httpRoundTripper
	err    error
}

func newExternalLinkDownloader(cfg *config.Config) *externalLinkDownloader {
	return &externalLinkDownloader{cfg: cfg}
}

func (d *externalLinkDownloader) ensureClient() error {
	if d.client != nil {
		return nil
	}
	d.once.Do(func() {
		if d.cfg == nil {
			d.err = errors.New("workspace configuration is not available")
			return
		}
		clientCfg, err := config.HTTPClientConfigFromConfig(d.cfg)
		if err != nil {
			d.err = err
			return
		}
		clientCfg.AuthVisitor = nil
		clientCfg.Visitors = nil
		clientCfg.AccountID = ""
		clientCfg.Host = ""
		d.client = httpclient.NewApiClient(clientCfg)
	})
	return d.err
}

func (d *externalLinkDownloader) download(ctx context.Context, link sqlapi.ExternalLink) ([][]string, error) {
	if err := d.ensureClient(); err != nil {
		return nil, err
	}
	if d.client == nil {
		return nil, errors.New("external link HTTP client not initialized")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link.ExternalLink, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range link.HttpHeaders {
		req.Header.Set(k, v)
	}
	resp, err := d.client.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("external link chunk %d returned %s", link.ChunkIndex, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var payload externalLinkPayload
	if err := json.Unmarshal(body, &payload); err == nil {
		switch {
		case len(payload.Result.DataArray) > 0:
			return payload.Result.DataArray, nil
		case len(payload.DataArray) > 0:
			return payload.DataArray, nil
		}
	}

	var rows [][]string
	if err := json.Unmarshal(body, &rows); err == nil {
		if len(rows) == 0 {
			return nil, fmt.Errorf("external link chunk %d returned no rows", link.ChunkIndex)
		}
		return rows, nil
	}

	return nil, fmt.Errorf("external link chunk %d returned unsupported payload", link.ChunkIndex)
}
