package flags

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/pflag"
)

// optionalValue is a flag bound to an optional (pointer) request field. The
// pointer stays nil unless the flag is set, so unset flags are not sent.
type optionalValue[T any] struct {
	p     **T
	typ   string
	parse func(string) (T, error)
}

func (v *optionalValue[T]) String() string {
	if *v.p == nil {
		switch v.typ {
		case "bool":
			return "false"
		case "string":
			return ""
		}
		return "0"
	}
	return fmt.Sprint(**v.p)
}

func (v *optionalValue[T]) Set(s string) error {
	x, err := v.parse(s)
	if err != nil {
		return err
	}
	*v.p = &x
	return nil
}

func (v *optionalValue[T]) Type() string { return v.typ }

type optionalBool struct{ optionalValue[bool] }

func (*optionalBool) IsBoolFlag() bool { return true }

// StringPtrVar defines a string flag that sets *p only when the flag is used.
func StringPtrVar(fs *pflag.FlagSet, p **string, name, usage string) {
	fs.Var(&optionalValue[string]{p: p, typ: "string", parse: func(s string) (string, error) { return s, nil }}, name, usage)
}

// BoolPtrVar defines a bool flag that sets *p only when the flag is used.
func BoolPtrVar(fs *pflag.FlagSet, p **bool, name, usage string) {
	fs.Var(&optionalBool{optionalValue[bool]{p: p, typ: "bool", parse: strconv.ParseBool}}, name, usage)
	fs.Lookup(name).NoOptDefVal = "true"
}

// IntPtrVar defines an int flag that sets *p only when the flag is used.
func IntPtrVar(fs *pflag.FlagSet, p **int, name, usage string) {
	fs.Var(&optionalValue[int]{p: p, typ: "int", parse: strconv.Atoi}, name, usage)
}

// Int64PtrVar defines an int64 flag that sets *p only when the flag is used.
func Int64PtrVar(fs *pflag.FlagSet, p **int64, name, usage string) {
	fs.Var(&optionalValue[int64]{p: p, typ: "int64", parse: func(s string) (int64, error) {
		return strconv.ParseInt(s, 0, 64)
	}}, name, usage)
}

// Float64PtrVar defines a float64 flag that sets *p only when the flag is used.
func Float64PtrVar(fs *pflag.FlagSet, p **float64, name, usage string) {
	fs.Var(&optionalValue[float64]{p: p, typ: "float64", parse: func(s string) (float64, error) {
		return strconv.ParseFloat(s, 64)
	}}, name, usage)
}

// enumValue binds a flag to a string enum and validates the given value.
type enumValue[T ~string] struct {
	p      *T
	typ    string
	values []string
}

func (v *enumValue[T]) String() string { return string(*v.p) }

func (v *enumValue[T]) Set(s string) error {
	if len(v.values) > 0 && !slices.Contains(v.values, s) {
		return fmt.Errorf(`value "%s" is not one of "%s"`, s, strings.Join(v.values, `", "`))
	}
	*v.p = T(s)
	return nil
}

func (v *enumValue[T]) Type() string { return v.typ }

// EnumVar defines a flag for a string enum. typ is shown in help output and
// values (when non-empty) are the accepted values.
func EnumVar[T ~string](fs *pflag.FlagSet, p *T, name, usage, typ string, values ...string) {
	fs.Var(&enumValue[T]{p: p, typ: typ, values: values}, name, usage)
}

// ParseEnum validates s against values (when non-empty) for a positional
// enum argument.
func ParseEnum[T ~string](s string, values ...string) (T, error) {
	if len(values) > 0 && !slices.Contains(values, s) {
		return "", fmt.Errorf(`value "%s" is not one of "%s"`, s, strings.Join(values, `", "`))
	}
	return T(s), nil
}
