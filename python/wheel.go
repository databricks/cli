package python

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"
	"strings"
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
