package flags

import (
	"fmt"
	"strings"
)

type ProgressLogFormat string

var ModeAppend = ProgressLogFormat("append")
var ModeInplace = ProgressLogFormat("inplace")
var ModeJson = ProgressLogFormat("json")
var ModeDefault = ProgressLogFormat("default")

func (p *ProgressLogFormat) String() string {
	return string(*p)
}

func NewProgressLogFormat() ProgressLogFormat {
	return ModeDefault
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
		return fmt.Errorf("accepted arguments are [%s]", strings.Join(valid, ", "))
	}
	return nil
}

func (p *ProgressLogFormat) Type() string {
	return "format"
}

// TODO: register autocomplete suggestions for cobra: https://github.com/databricks/bricks/issues/279
