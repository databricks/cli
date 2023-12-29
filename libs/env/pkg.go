package env

// The env package provides functions for working with environment variables
// and allowing for overrides via the context.Context. This is useful for
// testing where tainting a processes' environment is at odds with parallelism.
// Use of a context.Context to store variable overrides means tests can be
// parallelized without worrying about environment variable interference.
