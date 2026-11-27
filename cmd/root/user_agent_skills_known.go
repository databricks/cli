package root

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/skills"
)

// knownSkills lists every skill name Databricks has published, current and
// retired. Only these names are reported in skill/<name> user agent pairs.
//
// The skills-invoked value comes from whatever agent runs the CLI, and agents
// pass all the skills they loaded, not only Databricks ones. A third-party or
// customer-written skill name can identify a customer or describe their
// internal work, so it must never leave the machine. A prefix check is not
// enough: customers name their own skills "databricks-<something>" too.
//
// Sources:
//   - https://github.com/databricks/databricks-agent-skills: every SKILL.md
//     directory name across the repository's history, including skills that were
//     renamed (databricks -> databricks-core, databricks-bundles -> databricks-dabs)
//     or retired, so older installs keep reporting.
//   - The databricks-quickstart skill printed by `databricks quickstart --skill`.
//
// A skill that is not listed here is still reported when it is in the skills
// release bundled with the binary (see releasedSkills), so new skills need no
// edit here. Retired and renamed names must stay, because no current release
// lists them. It is a slice, not a map, so it costs nothing at startup.
var knownSkills = []string{
	"data-app-design",
	"databricks",
	"databricks-agent-bricks",
	"databricks-ai-functions",
	"databricks-ai-runtime",
	"databricks-aibi-dashboards",
	"databricks-app-design",
	"databricks-apps",
	"databricks-apps-python",
	"databricks-bundles",
	"databricks-cli",
	"databricks-cli-auth",
	"databricks-cli-install",
	"databricks-config",
	"databricks-core",
	"databricks-dabs",
	"databricks-data-discovery",
	"databricks-dbsql",
	"databricks-docs",
	"databricks-execution-compute",
	"databricks-genie",
	"databricks-genie-agents",
	"databricks-iceberg",
	"databricks-jobs",
	"databricks-lakebase",
	"databricks-lakebase-autoscale",
	"databricks-lakebase-provisioned",
	"databricks-lakeflow-connect",
	"databricks-lakeflow-connect-saas",
	"databricks-lakehouse-mcp",
	"databricks-metric-view-advisor",
	"databricks-metric-views",
	"databricks-ml-training",
	"databricks-mlflow-evaluation",
	"databricks-mlflow-tracing",
	"databricks-model-serving",
	"databricks-pipelines",
	"databricks-python-sdk",
	"databricks-quickstart",
	"databricks-sandbox",
	"databricks-serverless",
	"databricks-serverless-migration",
	"databricks-serverless-migration-api",
	"databricks-setup-local",
	"databricks-spark-declarative-pipelines",
	"databricks-spark-structured-streaming",
	"databricks-synthetic-data-gen",
	"databricks-unity-catalog",
	"databricks-unity-gateway",
	"databricks-unstructured-pdf-generation",
	"databricks-vector-search",
	"databricks-zerobus-ingest",
	"lakeflow-unstructured",
	"spark-python-data-source",
}

// releasedSkillsTimeout bounds the wait for releasedSkills in one parseSkills
// call. It is a variable so tests can change it.
var releasedSkillsTimeout = 5 * time.Millisecond

// releasedSkillSet holds the names of the skills release that ships with this
// binary. done is closed once names and err are set.
type releasedSkillSet struct {
	done  chan struct{}
	names []string
	err   error
}

// releasedSkills starts the lookup of the released skill names on the first
// call, once per process, so a command whose skills are all in knownSkills
// never pays for it. A variable so tests can swap the loader.
var releasedSkills = newReleasedSkills(loadBundledSkillNames)

func newReleasedSkills(load func() ([]string, error)) func() *releasedSkillSet {
	return sync.OnceValue(func() *releasedSkillSet {
		set := &releasedSkillSet{done: make(chan struct{})}
		go func() {
			set.names, set.err = load()
			close(set.done)
		}()
		return set
	})
}

// loadBundledSkillNames returns the skill names in the skills release compiled
// into the binary, which the release build fills from the published release.
// It only reads a source that can't name a customer's own skill and needs no
// network call: the bundled snapshot, not DATABRICKS_SKILLS_DIR or GitHub. A
// development build has an empty snapshot and returns an error.
func loadBundledSkillNames() ([]string, error) {
	// The lookup outlives a caller that stopped waiting at its deadline, so it
	// does not take the caller's context. Reading the embedded snapshot does no
	// I/O that a context could cancel.
	catalog, err := skills.NewBundledSource().Catalog(context.Background()) //nolint:gocritic // see above
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(catalog.Skills))
	for _, s := range catalog.Skills {
		names = append(names, s.Name)
	}
	return names, nil
}

// skillMatcher reports whether a name is a Databricks skill: one in knownSkills,
// or one in the released skills when their lookup finishes before the deadline.
// A lookup that misses the deadline counts as no match, so the name is dropped.
type skillMatcher struct {
	deadline   <-chan time.Time
	skipLookup bool
}

// newSkillMatcher returns a matcher for one parseSkills call. The deadline
// starts at the first name that is not in knownSkills and is shared by every
// later name, so a slow lookup costs at most releasedSkillsTimeout in total.
func newSkillMatcher() *skillMatcher {
	return &skillMatcher{}
}

func (m *skillMatcher) isKnown(ctx context.Context, name string) bool {
	if slices.Contains(knownSkills, name) {
		return true
	}
	if m.skipLookup {
		return false
	}
	if m.deadline == nil {
		m.deadline = time.After(releasedSkillsTimeout)
	}
	set := releasedSkills()
	select {
	case <-set.done:
		if set.err != nil {
			log.Debugf(ctx, "no released skills catalog for skill telemetry: %v", set.err)
			m.skipLookup = true
			return false
		}
		return slices.Contains(set.names, name)
	case <-m.deadline:
		m.skipLookup = true
		log.Debugf(ctx, "released skills lookup took longer than %s, dropping skills not in the built-in list", releasedSkillsTimeout)
		return false
	}
}
