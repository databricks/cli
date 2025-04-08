package internal

// Substitute env variables like $VAR in value.
// value is a string with variable references like $VAR
// env is a set of variables in golang format, like VAR=hello
// Example: value="$CLI", env={"CLI=/bin/true"}, result: "/bin/true"
func SubstituteEnv(value string, env []string) string {
	// AI TODO: implement this
}
