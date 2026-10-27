package filer

import (
	"context"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/databricks/sdk-go/core/clientinfo"
)

// filesAPIOwnedUserAgentKeys are segments that only describe the classic SDK
// or that the files/v2 client already renders on its own (from the
// environment or its credentials). Copying them would misreport or duplicate
// them.
var filesAPIOwnedUserAgentKeys = map[string]bool{
	"databricks-sdk-go": true,
	"go":                true,
	"os":                true,
	"auth":              true,
	"cicd":              true,
	"agent":             true,
	"runtime":           true,
	"meta-harness":      true,
}

// registerUserAgentForFilesAPI copies the segments the CLI attaches to the
// classic SDK context (cmd, cmd-exec-id, interactive, engine, ...) into the
// files/v2 client's process-wide user agent, so that files/v2 requests are
// attributed to the same command as the rest of the CLI's traffic.
//
// files/v2 has no request-scoped segments. A CLI process runs a single
// command, so process-wide segments carry the same information.
func registerUserAgentForFilesAPI(ctx context.Context) {
	rendered := map[string]bool{}
	for s := range strings.FieldsSeq(clientinfo.Default().String()) {
		rendered[s] = true
	}

	for s := range strings.FieldsSeq(useragent.FromContext(ctx)) {
		key, value, ok := strings.Cut(s, "/")
		if !ok || rendered[s] || filesAPIOwnedUserAgentKeys[key] {
			continue
		}
		if err := clientinfo.AddToDefault(key, value); err != nil {
			log.Debugf(ctx, "Not adding %q to the files/v2 user agent: %s", s, err)
		}
	}
}
