package flags

import (
	"fmt"
	"strings"
	"syscall"

	"golang.org/x/term"
)

type ProgressLogFormat string

var ModeAppend = ProgressLogFormat("append")
var ModeInplace = ProgressLogFormat("inplace")
var ModeJson = ProgressLogFormat("json")

func (p *ProgressLogFormat) String() string {
	return string(*p)
}

func NewProgressLogFormat() ProgressLogFormat {
	if term.IsTerminal(syscall.Stderr) {
		return ModeInplace
	}
	return ModeAppend
}

func (p *ProgressLogFormat) Set(s string) error {
	lower := strings.ToLower(s)
	switch lower {
	case ModeAppend.String():
		*p = ProgressLogFormat(ModeAppend.String())
	case ModeInplace.String():
		*p = ProgressLogFormat(ModeInplace.String())
	case ModeJson.String():
		*p = ProgressLogFormat(ModeJson.String())
	default:
		valid := []string{
			ModeAppend.String(),
			ModeInplace.String(),
			ModeJson.String(),
		}
		return fmt.Errorf("accepted arguments are %s", strings.Join(valid, " , "))
	}
	return nil
}

func (p *ProgressLogFormat) Type() string {
	return "format"
}
