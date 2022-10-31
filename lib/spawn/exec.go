package spawn

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"runtime"
	"strings"
)

type cxtKey int

const (
	prjRoot cxtKey = iota
)

func WithRoot(ctx context.Context, root string) context.Context {
	return context.WithValue(ctx, prjRoot, root)
}

func ExecAndPassErr(ctx context.Context, name string, args ...string) ([]byte, error) {
	log.Printf("[DEBUG] Running %s %s", name, strings.Join(args, " "))

	reader, writer := io.Pipe()

	out := bytes.NewBuffer([]byte{}) // add option to route to Stdout in verbose mode
	go io.Copy(out, reader)

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = writer
	cmd.Stderr = writer

	root, ok := ctx.Value(prjRoot).(string)
	if ok {
		cmd.Dir = root
	}

	err := cmd.Run()
	_ = writer.Close()
	_ = reader.Close()

	if err != nil {
		return nil, fmt.Errorf(trimmedS(out.Bytes()))
	}

	return []byte(trimmedS(out.Bytes())), nil
}

func DetectExecutable(ctx context.Context, exec string) (string, error) {
	detector := "which"
	if runtime.GOOS == "windows" {
		detector = "where.exe"
	}
	out, err := ExecAndPassErr(ctx, detector, exec)
	if err != nil {
		return "", err
	}
	return trimmedS(out), nil
}

func nicerErr(err error) error {
	if err == nil {
		return nil
	}
	if ee, ok := err.(*exec.ExitError); ok {
		errMsg := trimmedS(ee.Stderr)
		if errMsg == "" {
			errMsg = err.Error()
		}
		return errors.New(errMsg)
	}
	return err
}

func trimmedS(bytes []byte) string {
	return strings.Trim(string(bytes), "\n\r")
}
