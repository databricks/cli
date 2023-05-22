package root

import (
	"errors"
	"strconv"
)

// errExitFromVersion is returned when the version flag is set.
var errExitFromVersion = errors.New("exit from version flag")

type versionFlag struct {
}

func (v *versionFlag) Set(s string) error {
	b, err := strconv.ParseBool(s)
	if err != nil {
		return err
	}
	if b {
		// Return if this flag as soon as this flag gets set.
		// If no runnable command is specified, the root command's
		// hooks are not executed, and this is the only place to
		// detect whether the version flag is set.
		return errExitFromVersion
	}
	return nil
}

func (v *versionFlag) Type() string {
	return "bool"
}

func (v *versionFlag) String() string {
	return "false"
}

func (v *versionFlag) IsBoolFlag() bool {
	return true
}

func init() {
	var f versionFlag
	flag := RootCmd.PersistentFlags().VarPF(&f, "version", "v", "print version and exit")
	flag.NoOptDefVal = "true"
}
