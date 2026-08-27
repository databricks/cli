package process

import "time"

// processGroupGracePeriod bounds how long WithProcessGroup waits after the
// context is cancelled before escalating to SIGKILL. It mirrors the 10s grace
// period used elsewhere for subprocess termination (see cmd/ssh). It is
// a var, not a const, only so the escalation test can shorten it.
var processGroupGracePeriod = 10 * time.Second
