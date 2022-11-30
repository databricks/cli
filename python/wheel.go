package python

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"github.com/databricks/bricks/project"
)

func BuildWheel(ctx context.Context, dir string) (string, error) {
	defer chdirAndBack(dir)()
	// remove previous dist leak
	os.RemoveAll("dist")
	// remove all other irrelevant traces
	silentlyCleanupWheelFolder(".")
	// call simple wheel builder. we may need to pip install wheel as well
	out, err := Py(ctx, "setup.py", "bdist_wheel")
	if err != nil {
		return "", err
	}
	log.Printf("[DEBUG] Built wheel: %s", out)

	// and cleanup afterwards
	silentlyCleanupWheelFolder(".")

	wheel := silentChildWithSuffix("dist", ".whl")
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
	defer chdirAndBack(dir)()
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

	wsc := project.Get(ctx).WorkspacesClient()
	wf, err := os.Open(wheel)
	if err != nil {
		return "", err
	}
	defer wf.Close()
	// err = dbfs.Create(dbfsLoc, raw, true)
	err = wsc.Dbfs.Overwrite(ctx, dbfsLoc, wf)
	// TODO: maintain PEP503 compliance and update meta-files:
	// ${DBFSWheelLocation}/index.html and ${DBFSWheelLocation}/${NormalizedName}/index.html
	return dbfsLoc, err
}

func silentlyCleanupWheelFolder(dir string) {
	// there or not there - we don't care
	os.RemoveAll(path.Join(dir, "__pycache__"))
	os.RemoveAll(path.Join(dir, "build"))
	eggInfo := silentChildWithSuffix(dir, ".egg-info")
	if eggInfo == "" {
		return
	}
	os.RemoveAll(eggInfo)
}

func silentChildWithSuffix(dir, suffix string) string {
	f, err := os.Open(dir)
	if err != nil {
		log.Printf("[DEBUG] open dir %s: %s", dir, err)
		return ""
	}
	entries, err := f.ReadDir(0)
	if err != nil {
		log.Printf("[DEBUG] read dir %s: %s", dir, err)
		// todo: log
		return ""
	}
	for _, child := range entries {
		if !strings.HasSuffix(child.Name(), suffix) {
			continue
		}
		return path.Join(dir, child.Name())
	}
	return ""
}

func chdirAndBack(dir string) func() {
	wd, _ := os.Getwd()
	os.Chdir(dir)
	return func() {
		os.Chdir(wd)
	}
}
