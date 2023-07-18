package python

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/databricks/cli/libs"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/files"
)

func BuildWheel(ctx context.Context, dir string) (string, error) {
	defer libs.ChdirAndBack(dir)()
	// remove previous dist leak
	os.RemoveAll("dist")
	// remove all other irrelevant traces
	CleanupWheelFolder(".")
	// call simple wheel builder. we may need to pip install wheel as well
	out, err := Py(ctx, "setup.py", "bdist_wheel")
	if err != nil {
		return "", err
	}
	log.Debugf(ctx, "Built wheel: %s", out)

	// and cleanup afterwards
	CleanupWheelFolder(".")

	wheel := FindFileWithSuffixInPath("dist", ".whl")
	if wheel == "" {
		return "", fmt.Errorf("cannot find built wheel in %s", dir)
	}
	return path.Join(dir, wheel), nil
}

const DBFSWheelLocation = "dbfs:/FileStore/wheels/simple"

// TODO: research deeper if we make new data resource for terraform, like `databricks_latest_wheel` (preferred),
// or do we bypass the environment variable into terraform deployer. And make a decision.
//
// Whatever this method gets refactored to is intended to be used for two purposes:
//   - uploading project's wheel archives: one per project or one per project/developer, depending on isolation
//   - synchronising enterprise artifactories, jfrogs, azdo feeds, so that we fix the gap of private code artifact
//     repository integration.
func UploadWheelToDBFSWithPEP503(ctx context.Context, dir string) (string, error) {
	wheel, err := BuildWheel(ctx, dir)
	if err != nil {
		return "", err
	}
	defer libs.ChdirAndBack(dir)()
	dist, err := ReadDistribution(ctx)
	if err != nil {
		return "", err
	}
	// TODO: figure out wheel naming criteria for Soft project isolation to allow multiple
	// people workin on the same project to upload wheels and let them be deployed as independent jobs.
	// we should also consider multiple PEP503 index stacking: per enterprise, per project, per developer.
	// PEP503 indexes can be rolled out to clusters via checksummed global init script, that creates
	// a driver/worker `/etc/pip.conf` with FUSE-mounted file:///dbfs/FileStore/wheels/simple/..
	// extra index URLs. See more pointers at https://stackoverflow.com/q/30889494/277035
	dbfsLoc := fmt.Sprintf("%s/%s/%s", DBFSWheelLocation, dist.NormalizedName(), path.Base(wheel))

	wsc, err := databricks.NewWorkspaceClient(&databricks.Config{})
	if err != nil {
		return "", err
	}
	wf, err := os.Open(wheel)
	if err != nil {
		return "", err
	}
	defer wf.Close()
	h, err := wsc.Dbfs.Open(ctx, dbfsLoc, files.FileModeOverwrite|files.FileModeWrite)
	if err != nil {
		return "", err
	}
	_, err = io.Copy(h, wf)
	// TODO: maintain PEP503 compliance and update meta-files:
	// ${DBFSWheelLocation}/index.html and ${DBFSWheelLocation}/${NormalizedName}/index.html
	return dbfsLoc, err
}
