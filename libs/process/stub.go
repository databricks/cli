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
	reponseStub
	calls           []*exec.Cmd
	defaultCallback func(*exec.Cmd) error
	responses       map[string]reponseStub
}

func (s *processStub) WithDefaultOutput(output string) *processStub {
	s.reponseStub.stdout = output
	return s
}

func (s *processStub) WithDefaultFailure(err error) *processStub {
	s.reponseStub.err = err
	return s
}

func (s *processStub) WithDefaultCallback(cb func(cmd *exec.Cmd) error) *processStub {
	s.defaultCallback = cb
	return s
}

// WithStdoutFor predefines standard output response for a command. The first word
// in the command string is the executable name, and NOT the executable path.
// The following command would stub "2" output for "/usr/local/bin/echo 1" command:
//
//	stub.WithStdoutFor("echo 1", "2")
func (s *processStub) WithStdoutFor(command, out string) *processStub {
	s.responses[command] = reponseStub{
		stdout: out,
	}
	return s
}

// WithStdoutFor same as [WithStdoutFor], but for standard error
func (s *processStub) WithStderrFor(command, out string) *processStub {
	s.responses[command] = reponseStub{
		stderr: out,
	}
	return s
}

// WithFailureFor same as [WithStdoutFor], but for process failures
func (s *processStub) WithFailureFor(command string, err error) *processStub {
	s.responses[command] = reponseStub{
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

// LookupEnv returns a value from any of the triggered process environments
func (s *processStub) LookupEnv(key string) string {
	environment := s.CombinedEnvironment()
	return environment[key]
}

func (s *processStub) normCmd(v *exec.Cmd) string {
	// to reduce testing noise, we collect here only the deterministic binary basenames, e.g.
	// "/var/folders/bc/7qf8yghj6v14t40096pdcqy40000gp/T/tmp.03CAcYcbOI/python3" becomes "python3",
	// while still giving the possibility to customize process stubbing even further.
	// See [processStub.WithDefaultCallback]
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
	if s.reponseStub.stdout != "" {
		cmd.Stdout.Write([]byte(s.reponseStub.stdout))
	}
	return s.reponseStub.err
}
