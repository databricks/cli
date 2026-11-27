package autoinstall

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDecide(t *testing.T) {
	// eligible is a run of a released version never seen before, with the gate on, in the treatment arm.
	eligible := Inputs{
		Version: "1.30.0",
		Now:     time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		GateOn:  true,
		Bucket:  HoldoutPercent,
	}
	tests := []struct {
		name   string
		modify func(in *Inputs)
		want   Reason
	}{
		{name: "eligible", modify: func(in *Inputs) {}, want: ReasonRun},

		{name: "skipped command", modify: func(in *Inputs) { in.SkippedCommand = true }, want: ReasonSkippedCommand},
		{name: "Databricks Runtime", modify: func(in *Inputs) { in.OnRuntime = true }, want: ReasonOnRuntime},

		{name: "cache disabled", modify: func(in *Inputs) { in.CacheDisabled = true }, want: ReasonCacheDisabled},
		{name: "development build", modify: func(in *Inputs) { in.Version = "1.31.0-dev+abcdef123456" }, want: ReasonDevelopmentBuild},
		{name: "snapshot build", modify: func(in *Inputs) { in.Version = "1.30.1-dev+abcdef1" }, want: ReasonDevelopmentBuild},
		{name: "root in a user's home", modify: func(in *Inputs) { in.RootInUserHome = true }, want: ReasonRootInUserHome},
		{name: "root in a user's home before the state", modify: func(in *Inputs) {
			in.RootInUserHome = true
			in.State = State{LastVersion: "1.30.0", Attempts: MaxAttempts}
			in.CI = true
		}, want: ReasonRootInUserHome},

		{name: "same version, settled", modify: func(in *Inputs) { in.State.LastVersion = "1.30.0" }, want: ReasonAlreadyRan},
		{name: "same version, failed before", modify: func(in *Inputs) { in.State = State{LastVersion: "1.30.0", Attempts: 3} }, want: ReasonRun},
		{name: "same version, last attempt left", modify: func(in *Inputs) { in.State = State{LastVersion: "1.30.0", Attempts: MaxAttempts - 1} }, want: ReasonRun},
		{name: "same version, retries exhausted", modify: func(in *Inputs) { in.State = State{LastVersion: "1.30.0", Attempts: MaxAttempts} }, want: ReasonRetriesExhausted},
		{name: "same version, failed just now", modify: func(in *Inputs) {
			in.State = State{LastVersion: "1.30.0", Attempts: 1, LastAttempt: in.Now}
		}, want: ReasonRetryNotDue},
		{name: "same version, retry due", modify: func(in *Inputs) {
			in.State = State{LastVersion: "1.30.0", Attempts: 1, LastAttempt: in.Now.Add(-firstRetryDelay)}
		}, want: ReasonRun},
		{name: "same version, retries exhausted while waiting", modify: func(in *Inputs) {
			in.State = State{LastVersion: "1.30.0", Attempts: MaxAttempts, LastAttempt: in.Now}
		}, want: ReasonRetriesExhausted},
		{name: "CI before the retry wait", modify: func(in *Inputs) {
			in.State = State{LastVersion: "1.30.0", Attempts: 1, LastAttempt: in.Now}
			in.CI = true
		}, want: ReasonSkippedCI},
		{name: "opt-out before the retry wait", modify: func(in *Inputs) {
			in.State = State{LastVersion: "1.30.0", Attempts: 1, LastAttempt: in.Now, OptedOut: true}
		}, want: ReasonSkippedOptOut},
		{name: "retry wait before the gate", modify: func(in *Inputs) {
			in.State = State{LastVersion: "1.30.0", Attempts: 1, LastAttempt: in.Now}
			in.GateOn = false
		}, want: ReasonRetryNotDue},
		{name: "new version while the old one waits", modify: func(in *Inputs) {
			in.State = State{LastVersion: "1.29.0", Attempts: 1, LastAttempt: in.Now}
		}, want: ReasonRun},
		{name: "new version after retries exhausted", modify: func(in *Inputs) { in.State = State{LastVersion: "1.29.0", Attempts: MaxAttempts} }, want: ReasonRun},
		{name: "older CLI after a newer one ran", modify: func(in *Inputs) { in.State.LastVersion = "1.31.0" }, want: ReasonAlreadyRan},
		{name: "older CLI while a newer one retries", modify: func(in *Inputs) { in.State = State{LastVersion: "1.31.0", Attempts: 2} }, want: ReasonAlreadyRan},
		{name: "newer CLI after an older one ran", modify: func(in *Inputs) { in.State.LastVersion = "1.29.5" }, want: ReasonRun},

		{name: "CI", modify: func(in *Inputs) { in.CI = true }, want: ReasonSkippedCI},
		{name: "CI before the opt-out", modify: func(in *Inputs) { in.CI = true; in.EnvOptOut = true }, want: ReasonSkippedCI},
		{name: "env opt-out", modify: func(in *Inputs) { in.EnvOptOut = true }, want: ReasonSkippedOptOut},
		{name: "persisted opt-out", modify: func(in *Inputs) { in.State.OptedOut = true }, want: ReasonSkippedOptOut},
		{name: "env opt-out over a stored consent", modify: func(in *Inputs) { in.EnvOptOut = true; in.State.Consented = true }, want: ReasonSkippedOptOut},
		{name: "opt-out before the gate", modify: func(in *Inputs) { in.State.OptedOut = true; in.GateOn = false }, want: ReasonSkippedOptOut},
		{name: "gate off", modify: func(in *Inputs) { in.GateOn = false }, want: ReasonGateOff},
		{name: "gate off hides the holdout", modify: func(in *Inputs) { in.GateOn = false; in.Bucket = 0 }, want: ReasonGateOff},
		{name: "holdout", modify: func(in *Inputs) { in.Bucket = HoldoutPercent - 1 }, want: ReasonSkippedHoldout},
		{name: "first bucket is holdout", modify: func(in *Inputs) { in.Bucket = 0 }, want: ReasonSkippedHoldout},
		{name: "last bucket is treatment", modify: func(in *Inputs) { in.Bucket = 99 }, want: ReasonRun},

		{name: "skipped command before everything", modify: func(in *Inputs) {
			in.SkippedCommand = true
			in.OnRuntime = true
			in.CacheDisabled = true
			in.CI = true
		}, want: ReasonSkippedCommand},
		{name: "settled version before CI", modify: func(in *Inputs) { in.State.LastVersion = "1.30.0"; in.CI = true }, want: ReasonAlreadyRan},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := eligible
			tc.modify(&in)
			assert.Equal(t, tc.want, Decide(in))
		})
	}
}

func TestRetryDue(t *testing.T) {
	last := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	tests := []struct {
		name     string
		attempts int
		since    time.Duration
		want     bool
	}{
		{name: "right after the first", attempts: 1, since: 0, want: false},
		{name: "before the first wait ends", attempts: 1, since: 5*time.Minute - time.Second, want: false},
		{name: "after the first wait", attempts: 1, since: 5 * time.Minute, want: true},
		{name: "the second wait doubles", attempts: 2, since: 10*time.Minute - time.Second, want: false},
		{name: "after the second wait", attempts: 2, since: 10 * time.Minute, want: true},
		{name: "the last wait", attempts: MaxAttempts - 1, since: 5 * time.Minute << (MaxAttempts - 2), want: true},
		{name: "before the last wait ends", attempts: MaxAttempts - 1, since: 5*time.Minute<<(MaxAttempts-2) - time.Second, want: false},
		{name: "last attempt in the future", attempts: 1, since: -time.Second, want: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := State{Attempts: tc.attempts, LastAttempt: last}
			assert.Equal(t, tc.want, retryDue(s, last.Add(tc.since)))
		})
	}

	// A state an older CLI wrote has no last attempt.
	assert.True(t, retryDue(State{Attempts: 3}, last))
}

func TestReasonDue(t *testing.T) {
	notDue := []Reason{ReasonSkippedCommand, ReasonOnRuntime, ReasonCacheDisabled, ReasonDevelopmentBuild, ReasonRootInUserHome, ReasonAlreadyRan, ReasonRetriesExhausted, ReasonRetryNotDue}
	due := []Reason{ReasonSkippedCI, ReasonSkippedOptOut, ReasonGateOff, ReasonSkippedHoldout, ReasonRun}
	for _, r := range notDue {
		assert.False(t, r.Due(), r)
	}
	for _, r := range due {
		assert.True(t, r.Due(), r)
	}
}
