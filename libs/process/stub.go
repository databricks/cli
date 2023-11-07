package process

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

var stubKey int

// WithStub creates process stub for fast and flexible testing of subprocesses
//
//	ctx := context.Background()
//	ctx, stub := WithStub(ctx)
//	stub.WithDefaultOutput("meeee")
//	out, err := Background(ctx, []string{"/usr/local/bin/meeecho", "1", "--foo", "bar"})
//	require.NoError(t, err)
//	require.Equal(t, "meeee", out)
//	require.Equal(t, 1, stub.Len())
//	require.Equal(t, []string{"meeecho 1 --foo bar"}, stub.Commands())
func WithStub(ctx context.Context) (context.Context, *processStub) {
	stub := &processStub{responses: map[string]reponseStub{}}
	ctx = context.WithValue(ctx, &stubKey, stub)
	return ctx, stub
}

func runCmd(ctx context.Context, cmd *exec.Cmd) error {
	stub, ok := ctx.Value(&stubKey).(*processStub)
	if ok {
		return stub.run(cmd)
	}
	return cmd.Run()
}

type reponseStub struct {
	stdout string
	stderr string
	err    error
}

type processStub struct {
	calls           []*exec.Cmd
	defaultOutput   string
	defaultFailure  error
	defaultCallback func(*exec.Cmd) error
	responses       map[string]reponseStub
}

func (s *processStub) WithDefaultOutput(output string) *processStub {
	s.defaultOutput = output
	return s
}

func (s *processStub) WithDefaultFailure(err error) *processStub {
	s.defaultFailure = err
	return s
}

func (s *processStub) WithDefaultCallback(cb func(cmd *exec.Cmd) error) *processStub {
	s.defaultCallback = cb
	return s
}

func (s *processStub) WithStdoutFor(norm, out string) *processStub {
	s.responses[norm] = reponseStub{
		stdout: out,
	}
	return s
}

func (s *processStub) WithStderrFor(norm, out string) *processStub {
	s.responses[norm] = reponseStub{
		stderr: out,
	}
	return s
}

func (s *processStub) WithFailureFor(norm string, err error) *processStub {
	s.responses[norm] = reponseStub{
		err: err,
	}
	return s
}

func (s *processStub) String() string {
	return fmt.Sprintf("process stub with %d calls", s.Len())
}

func (s *processStub) Len() int {
	return len(s.calls)
}

func (s *processStub) Commands() (called []string) {
	for _, v := range s.calls {
		called = append(called, s.normCmd(v))
	}
	return
}

// CombinedEnvironment returns all enviroment variables used for all commands
//
//	ctx := context.Background()
//	ctx = env.Set(ctx, "FOO", "bar")
//	ctx, stub := WithStub(ctx)
//	out, err := Background(ctx, []string{"/usr/local/bin/meeecho", "1", "--foo", "bar"})
//	require.NoError(t, err)
//	allEnv := stub.CombinedEnvironment()
//	require.Equal(t, "bar", allEnv["FOO"])
//	require.Equal(t, "bar", stub.LookupEnv("FOO"))
func (s *processStub) CombinedEnvironment() map[string]string {
	environment := map[string]string{}
	for _, cmd := range s.calls {
		for _, line := range cmd.Env {
			k, v, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			environment[k] = v
		}
	}
	return environment
}

// CombinedEnvironment returns all enviroment variables used for all commands
//
//	ctx := context.Background()
//	ctx = env.Set(ctx, "FOO", "bar")
//	ctx, stub := WithStub(ctx)
//	out, err := Background(ctx, []string{"/usr/local/bin/meeecho", "1", "--foo", "bar"})
//	require.NoError(t, err)
//	require.Equal(t, "bar", stub.LookupEnv("FOO"))
func (s *processStub) LookupEnv(key string) string {
	environment := s.CombinedEnvironment()
	return environment[key]
}

func (s *processStub) normCmd(v *exec.Cmd) string {
	// to reduce testing noise, we collect here only the deterministic binary basenames, e.g.
	// "/var/folders/bc/7qf8yghj6v14t40096pdcqy40000gp/T/tmp.03CAcYcbOI/python3" becomes "python3",
	// while still giving the possibility to customize. See [processStub.WithDefaultCallback]
	binaryName := filepath.Base(v.Path)
	args := strings.Join(v.Args[1:], " ")
	return fmt.Sprintf("%s %s", binaryName, args)
}

func (s *processStub) run(cmd *exec.Cmd) error {
	s.calls = append(s.calls, cmd)
	resp, ok := s.responses[s.normCmd(cmd)]
	if ok {
		if resp.stdout != "" {
			cmd.Stdout.Write([]byte(resp.stdout))
		}
		if resp.stderr != "" {
			cmd.Stderr.Write([]byte(resp.stderr))
		}
		return resp.err
	}
	if s.defaultCallback != nil {
		return s.defaultCallback(cmd)
	}
	if s.defaultOutput != "" {
		cmd.Stdout.Write([]byte(s.defaultOutput))
	}
	return s.defaultFailure
}
