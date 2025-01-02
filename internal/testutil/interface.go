package testutil

// TestingT is an interface wrapper around *testing.T that provides the methods
// that are used by the test package to convey information about test failures.
//
// We use an interface so we can wrap *testing.T and provide additional functionality.
type TestingT interface {
	Log(args ...any)
	Logf(format string, args ...any)

	Error(args ...any)
	Errorf(format string, args ...any)

	Fatal(args ...any)
	Fatalf(format string, args ...any)

	Skip(args ...any)
	Skipf(format string, args ...any)

	FailNow()

	Cleanup(func())

	Setenv(key, value string)

	TempDir() string

	Helper()
}
