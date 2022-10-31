package mvn

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/databricks/bricks/lib/flavor"
	"github.com/databricks/bricks/lib/spawn"
	"github.com/databricks/databricks-sdk-go/service/libraries"
)

type Pom struct {
	Name       string `xml:"name"`
	GroupID    string `xml:"groupId"`
	ArtifactID string `xml:"artifactId"`
	Version    string `xml:"version"`
}

func (pom *Pom) Jar() string {
	return fmt.Sprintf("%s-%s.jar", pom.ArtifactID, pom.Version)
}

type Maven struct {
	SkipTests bool `json:"skip_tests,omitempty"`
}

func (mvn *Maven) RequiresCluster() bool {
	return true
}

// Java libraries always require cluster restart
func (mvn *Maven) RequiresRestart() bool {
	return true
}

func (mvn *Maven) Detected(prj flavor.Project) bool {
	_, err := os.Stat(filepath.Join(prj.Root(), "pom.xml"))
	return err == nil
}

func (mvn *Maven) LocalArtifacts(ctx context.Context, prj flavor.Project) (flavor.Artifacts, error) {
	pom, err := mvn.Pom(prj.Root())
	if err != nil {
		return nil, err
	}
	return flavor.Artifacts{
		{
			Flavor: mvn,
			Library: libraries.Library{
				Jar: fmt.Sprintf("%s/target/%s", prj.Root(), pom.Jar()),
			},
		},
	}, nil
}

func (mvn *Maven) Pom(root string) (*Pom, error) {
	// TODO: perhaps we should call effective-pom, specially once
	// we start comparing local spark version and the one on DBR
	pomFile := fmt.Sprintf("%s/pom.xml", root)
	pomHandle, err := os.Open(pomFile)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", pomFile, err)
	}
	pomBytes, err := io.ReadAll(pomHandle)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", pomFile, err)
	}
	var pom Pom
	err = xml.Unmarshal(pomBytes, &pom)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", pomFile, err)
	}
	return &pom, nil
}

func (mvn *Maven) Build(ctx context.Context, prj flavor.Project, status func(string)) error {
	mavenPath, err := spawn.DetectExecutable(ctx, "mvn")
	if err != nil {
		return fmt.Errorf("no Maven installed: %w", err)
	}
	// report back the name of the JAR to the user
	pom, _ := mvn.Pom(prj.Root())
	status(fmt.Sprintf("Buidling %s", pom.Jar()))
	args := []string{fmt.Sprintf("--file=%s/pom.xml", prj.Root())}
	args = append(args, "-DskipTests=true")
	args = append(args, "clean", "package")
	_, err = spawn.ExecAndPassErr(ctx, mavenPath, args...)
	if err != nil {
		// TODO: figure out error reporting in a generic way
		// one of the options is to re-run the same command with stdout forwarding
		return fmt.Errorf("mvn package: %w", err)
	}
	return nil
}
