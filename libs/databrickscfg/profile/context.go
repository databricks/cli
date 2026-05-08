package profile

import "context"

var profiler int

func GetProfiler(ctx context.Context) Profiler {
	p, ok := ctx.Value(&profiler).(Profiler)
	if !ok {
		return DefaultProfiler
	}
	return p
}
