// Package skillsinvoked sends the agent skills that drove a CLI call to
// Databricks in the X-Databricks-Skills-Invoked header.
package skillsinvoked

import (
	"context"
	"net/http"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/config"
)

// Header is the request header that lists the skills that drove the call.
const Header = "X-Databricks-Skills-Invoked"

const (
	maxSkills      = 20
	maxSkillLength = 64
)

// skillNamePattern matches plugin-scoped or plain skill names such as
// "databricks-core" or "databricks:databricks-core".
var skillNamePattern = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^[a-z0-9][a-z0-9_.:-]*$`)
})

type skillsKey struct{}

// Parse normalizes a comma-separated list of skill names. Entries are trimmed
// and lowercased, and empty, invalid, or over-long entries are dropped. Duplicates
// are removed and the list is capped at maxSkills. Dropped entries are logged at
// debug level only, because telemetry must never fail or warn a command.
func Parse(ctx context.Context, raw string) []string {
	var skills []string
	for entry := range strings.SplitSeq(raw, ",") {
		name := strings.ToLower(strings.TrimSpace(entry))
		switch {
		case name == "":
		case len(name) > maxSkillLength || !skillNamePattern().MatchString(name):
			log.Debugf(ctx, "dropping invalid skills-invoked entry %q", entry)
		case slices.Contains(skills, name):
		case len(skills) == maxSkills:
			log.Debugf(ctx, "dropping skills-invoked entry %q: more than %d skills", name, maxSkills)
		default:
			skills = append(skills, name)
		}
	}
	return skills
}

// WithSkills returns a context that carries the skills that drove the command.
func WithSkills(ctx context.Context, skills []string) context.Context {
	return context.WithValue(ctx, skillsKey{}, skills)
}

// FromContext returns the skills stored by WithSkills, or nil if there are none.
func FromContext(ctx context.Context) []string {
	skills, _ := ctx.Value(skillsKey{}).([]string)
	return skills
}

// Apply sets the header on every request made by cfg. The header is read from
// each request's context, so the command context must reach the request. A
// header hook already set on cfg (for example by liteswap) keeps running first.
func Apply(cfg *config.Config) {
	prev := cfg.Headers
	cfg.Headers = func(r *http.Request) error {
		if prev != nil {
			if err := prev(r); err != nil {
				return err
			}
		}
		setHeader(r)
		return nil
	}
}

// Transport wraps base so that it sets the header on requests whose context
// carries skills. Use it for clients that do not go through config.Config.Headers.
func Transport(base http.RoundTripper) http.RoundTripper {
	return roundTripper{base: base}
}

type roundTripper struct {
	base http.RoundTripper
}

func (t roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if len(FromContext(r.Context())) > 0 {
		r = r.Clone(r.Context())
		setHeader(r)
	}
	return t.base.RoundTrip(r)
}

// setHeader sets the header on r when its context carries skills.
func setHeader(r *http.Request) {
	if skills := FromContext(r.Context()); len(skills) > 0 {
		r.Header.Set(Header, strings.Join(skills, ","))
	}
}
