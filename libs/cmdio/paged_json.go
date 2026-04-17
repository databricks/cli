package cmdio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/databricks/databricks-sdk-go/listing"
)

// renderIteratorPagedJSON streams the iterator as a JSON array, pausing
// for user input every pagerPageSize items. The output is always a
// syntactically valid JSON array — even when the user quits early, we
// still write the closing ']' before returning so a reader that pipes
// the output to a file gets parseable JSON.
//
// The caller is expected to have already verified that the terminal
// supports paging (Capabilities.SupportsPager).
func renderIteratorPagedJSON[T any](
	ctx context.Context,
	iter listing.Iterator[T],
	out io.Writer,
) error {
	keys, restore, err := startRawStdinKeyReader(ctx)
	if err != nil {
		return err
	}
	defer restore()
	return renderIteratorPagedJSONCore(
		ctx,
		iter,
		crlfWriter{w: out},
		crlfWriter{w: os.Stderr},
		keys,
		pagerPageSize,
	)
}

// renderIteratorPagedJSONCore is the testable core of
// renderIteratorPagedJSON: it takes the output streams and key channel
// as dependencies and never touches os.Stdin directly.
//
// The rendering mirrors iteratorRenderer.renderJson — a pretty-printed
// JSON array with 2-space indentation — but Flush() is followed by a
// user prompt once every pageSize items. If the user says to quit, the
// closing bracket is still written so the accumulated output remains a
// valid JSON document.
func renderIteratorPagedJSONCore[T any](
	ctx context.Context,
	iter listing.Iterator[T],
	out io.Writer,
	prompts io.Writer,
	keys <-chan byte,
	pageSize int,
) error {
	if pageSize <= 0 {
		pageSize = pagerPageSize
	}

	// We render into an intermediate buffer and flush at page
	// boundaries. This lets us show the user N items, prompt, then
	// continue — without juggling partial writes to the underlying
	// writer mid-item.
	var buf bytes.Buffer
	flush := func() error {
		if buf.Len() == 0 {
			return nil
		}
		if _, err := out.Write(buf.Bytes()); err != nil {
			return err
		}
		buf.Reset()
		return nil
	}

	// We defer writing the opening bracket until we actually have an
	// item to write. That way an empty iterator (or one that errors
	// before yielding anything) produces "[]\n" — valid JSON — rather
	// than a half-open "[\n  " that the caller can't parse.
	totalWritten := 0
	finalize := func() error {
		if totalWritten == 0 {
			buf.WriteString("[]\n")
		} else {
			buf.WriteString("\n]\n")
		}
		return flush()
	}

	limit := limitFromContext(ctx)
	drainAll := false
	inPage := 0

	for iter.HasNext(ctx) {
		if limit > 0 && totalWritten >= limit {
			break
		}
		item, err := iter.Next(ctx)
		if err != nil {
			_ = finalize()
			return err
		}
		if totalWritten == 0 {
			buf.WriteString("[\n  ")
		} else {
			buf.WriteString(",\n  ")
		}
		encoded, err := json.MarshalIndent(item, "  ", "  ")
		if err != nil {
			_ = finalize()
			return err
		}
		buf.Write(encoded)
		totalWritten++
		inPage++

		if inPage < pageSize {
			continue
		}
		// End of a page. Flush what we have and either prompt or
		// continue the drain.
		if err := flush(); err != nil {
			return err
		}
		inPage = 0
		if drainAll {
			if pagerShouldQuit(keys) {
				return finalize()
			}
			continue
		}
		fmt.Fprint(prompts, pagerPromptText)
		k, ok := pagerNextKey(ctx, keys)
		fmt.Fprint(prompts, pagerClearLine)
		if !ok {
			return finalize()
		}
		switch k {
		case ' ':
			// continue with the next page
		case '\r', '\n':
			drainAll = true
		case 'q', 'Q', pagerKeyEscape, pagerKeyCtrlC:
			return finalize()
		}
	}
	return finalize()
}
