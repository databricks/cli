package profile

import (
	"context"
)

type ProfileMatchFunction func(Profile) bool

func MatchWorkspaceProfiles(p Profile) bool {
	return p.AccountID == ""
}

func MatchAccountProfiles(p Profile) bool {
	return p.Host != "" && p.AccountID != ""
}

func MatchAllProfiles(p Profile) bool {
	return true
}

func WithName(name string) ProfileMatchFunction {
	return func(p Profile) bool {
		return p.Name == name
	}
}

type Profiler interface {
	LoadProfiles(context.Context, ProfileMatchFunction) (Profiles, error)
	GetPath(context.Context) (string, error)
}

var DefaultProfiler = FileProfilerImpl{}
