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

// Header lists the skills that drove the call.
const Header = "X-Databricks-Skills-Invoked"

const (
	maxSkills      = 20
	maxSkillLength = 64
)

// skillNamePattern also allows plugin-scoped names such as "databricks:databricks-core".
var skillNamePattern = sync.OnceValue(func() *regexp.Regexp {
	return regexp.MustCompile(`^[a-z0-9][a-z0-9_.:-]*$`)
})

type skillsKey struct{}

// Parse normalizes a comma-separated list of skill names: entries are trimmed,
// lowercased and deduplicated, invalid ones are dropped, and at most maxSkills
// are kept. Invalid input is only logged at debug level because this is
// telemetry and must never fail or warn a command.
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

// FromContext returns the skills stored by WithSkills.
func FromContext(ctx context.Context) []string {
	skills, _ := ctx.Value(skillsKey{}).([]string)
	return skills
}

// Apply sets the header on every request made with cfg, keeping any header
// hook already on cfg (such as liteswap's). The value comes from the request
// context, so it is only sent when the command context reaches the request.
func Apply(cfg *config.Config) {
	prev := cfg.Headers
	cfg.Headers = func(r *http.Request) error {
		if prev != nil {
			if err := prev(r); err != nil {
				return err
			}
		}
		if skills := FromContext(r.Context()); len(skills) > 0 {
			r.Header.Set(Header, strings.Join(skills, ","))
		}
		return nil
	}
}

// Transport wraps base to set the header, for clients that don't use config.Config.Headers.
func Transport(base http.RoundTripper) http.RoundTripper {
	return roundTripper{base: base}
}

type roundTripper struct {
	base http.RoundTripper
}

func (t roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	if skills := FromContext(r.Context()); len(skills) > 0 {
		// A RoundTripper must not modify the caller's request.
		r = r.Clone(r.Context())
		r.Header.Set(Header, strings.Join(skills, ","))
	}
	return t.base.RoundTrip(r)
}
