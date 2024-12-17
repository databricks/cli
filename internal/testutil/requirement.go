package testutil

// Requirement is the interface for test requirements.
type Requirement interface {
	Verify(t TestingT)
}

// Require should be called at the beginning of a test to ensure that all
// requirements are met before running the test.
// If any requirement is not met, the test will be skipped.
func Require(t TestingT, requirements ...Requirement) {
	for _, r := range requirements {
		r.Verify(t)
	}
}
