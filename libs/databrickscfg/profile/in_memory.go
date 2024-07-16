package profile

import "context"

type InMemoryProfiler struct {
	Profiles Profiles
}

// GetPath implements Profiler.
func (i InMemoryProfiler) GetPath(context.Context) (string, error) {
	return "<in memory>", nil
}

// LoadProfiles implements Profiler.
func (i InMemoryProfiler) LoadProfiles(ctx context.Context, f ProfileMatchFunction) (Profiles, error) {
	res := make(Profiles, 0)
	for _, p := range i.Profiles {
		if f(p) {
			res = append(res, p)
		}
	}
	return res, nil
}

var _ Profiler = InMemoryProfiler{}
