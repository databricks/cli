package process

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
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
	calls     []*exec.Cmd
	callback  func(*exec.Cmd) error
	responses map[string]reponseStub
}

func (s *processStub) WithStdout(output string) *processStub {
	s.reponseStub.stdout = output
	return s
}

func (s *processStub) WithFailure(err error) *processStub {
	s.reponseStub.err = err
	return s
}

func (s *processStub) WithCallback(cb func(cmd *exec.Cmd) error) *processStub {
	s.callback = cb
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
		stderr: s.responses[command].stderr,
		err:    s.responses[command].err,
	}
	return s
}

// WithStderrFor same as [WithStdoutFor], but for standard error
func (s *processStub) WithStderrFor(command, out string) *processStub {
	s.responses[command] = reponseStub{
		stderr: out,
		stdout: s.responses[command].stdout,
		err:    s.responses[command].err,
	}
	return s
}

// WithFailureFor same as [WithStdoutFor], but for process failures
func (s *processStub) WithFailureFor(command string, err error) *processStub {
	s.responses[command] = reponseStub{
		err:    err,
		stderr: s.responses[command].stderr,
		stdout: s.responses[command].stdout,
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
	// "/var/folders/bc/7qf8yghj6v14t40096pdcqy40000gp/T/tmp.03CAcYcbOI/python3" becomes "python3".
	// Use [processStub.WithCallback] if you need to match against the full executable path.
	binaryName := filepath.Base(v.Path)
	var unixArgs []string
	for _, arg := range v.Args[1:] {
		unixArgs = append(unixArgs, filepath.ToSlash(arg))
	}
	args := strings.Join(unixArgs, " ")
	return fmt.Sprintf("%s %s", binaryName, args)
}

func (s *processStub) run(cmd *exec.Cmd) error {
	s.calls = append(s.calls, cmd)
	for pattern, resp := range s.responses {
		re := regexp.MustCompile(pattern)
		norm := s.normCmd(cmd)
		if !re.MatchString(norm) {
			continue
		}
		err := resp.err
		if resp.stdout != "" {
			_, err1 := cmd.Stdout.Write([]byte(resp.stdout))
			if err == nil {
				err = err1
			}
		}
		if resp.stderr != "" {
			_, err1 := cmd.Stderr.Write([]byte(resp.stderr))
			if err == nil {
				err = err1
			}
		}
		return err
	}
	if s.callback != nil {
		return s.callback(cmd)
	}
	var zeroStub reponseStub
	if s.reponseStub == zeroStub {
		return errors.New("no default process stub")
	}
	err := s.reponseStub.err
	if s.reponseStub.stdout != "" {
		_, err1 := cmd.Stdout.Write([]byte(s.reponseStub.stdout))
		if err == nil {
			err = err1
		}
	}
	return err
}
