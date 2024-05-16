package profile

import "context"

var profiler int

func WithProfiler(ctx context.Context, p Profiler) context.Context {
	return context.WithValue(ctx, &profiler, p)
}

func GetProfiler(ctx context.Context) Profiler {
	p, ok := ctx.Value(&profiler).(Profiler)
	if !ok {
		return DefaultProfiler
	}
	return p
}
