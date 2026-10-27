package filer

import (
	"context"
	"strings"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/databricks/sdk-go/core/clientinfo"
)

// filesAPIOwnedUserAgentKeys are classic-SDK-only segments or segments that
// files/v2 renders itself, so copying them would misreport or duplicate them.
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

// registerUserAgentForFilesAPI copies the CLI's classic SDK context segments
// (cmd, engine, ...) into the files/v2 user agent. files/v2 only supports
// process-wide segments, which is equivalent for a single-command process.
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
