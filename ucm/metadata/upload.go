package metadata

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/deploy"
	"github.com/databricks/cli/ucm/deploy/filer"
)

// MetadataFileName is the remote basename of the ucm metadata blob. It sits
// beside ucm-state.json in the same remote state dir so the two travel
// together; the ucm- prefix disambiguates from bundle's metadata.json for
// workspaces that host both kinds of deployment.
const MetadataFileName = "ucm-metadata.json"

// Upload writes md to <StateFiler root>/ucm-metadata.json. The caller is
// responsible for deciding whether an upload failure should be fatal; in the
// deploy phase we treat it as a warning because the deploy already succeeded
// by the time Upload runs.
func Upload(ctx context.Context, u *ucm.Ucm, backend deploy.Backend, md Metadata) error {
	if u == nil {
		return errors.New("ucm metadata: Upload called with nil Ucm")
	}
	if backend.StateFiler == nil {
		return errors.New("ucm metadata: Upload requires StateFiler in Backend")
	}

	blob, err := json.MarshalIndent(md, "", "  ")
	if err != nil {
		return err
	}
	return backend.StateFiler.Write(ctx, MetadataFileName, bytes.NewReader(blob), filer.WriteModeOverwrite|filer.WriteModeCreateParents)
}
