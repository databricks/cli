package autoinstall

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecide(t *testing.T) {
	// eligible is a run of a released version never seen before, with the gate on, in the treatment arm.
	eligible := Inputs{
		Version: "1.30.0",
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
		{name: "new version after retries exhausted", modify: func(in *Inputs) { in.State = State{LastVersion: "1.29.0", Attempts: MaxAttempts} }, want: ReasonRun},
		{name: "older CLI after a newer one ran", modify: func(in *Inputs) { in.State.LastVersion = "1.31.0" }, want: ReasonAlreadyRan},
		{name: "older CLI while a newer one retries", modify: func(in *Inputs) { in.State = State{LastVersion: "1.31.0", Attempts: 2} }, want: ReasonAlreadyRan},
		{name: "newer CLI after an older one ran", modify: func(in *Inputs) { in.State.LastVersion = "1.29.5" }, want: ReasonRun},

		{name: "CI", modify: func(in *Inputs) { in.CI = true }, want: ReasonSkippedCI},
		{name: "CI before the opt-out", modify: func(in *Inputs) { in.CI = true; in.EnvOptOut = true }, want: ReasonSkippedCI},
		{name: "env opt-out", modify: func(in *Inputs) { in.EnvOptOut = true }, want: ReasonSkippedOptOut},
		{name: "persisted opt-out", modify: func(in *Inputs) { in.State.OptedOut = true }, want: ReasonSkippedOptOut},
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

func TestReasonDue(t *testing.T) {
	notDue := []Reason{ReasonSkippedCommand, ReasonOnRuntime, ReasonCacheDisabled, ReasonDevelopmentBuild, ReasonRootInUserHome, ReasonAlreadyRan, ReasonRetriesExhausted}
	due := []Reason{ReasonSkippedCI, ReasonSkippedOptOut, ReasonGateOff, ReasonSkippedHoldout, ReasonRun}
	for _, r := range notDue {
		assert.False(t, r.Due(), r)
	}
	for _, r := range due {
		assert.True(t, r.Due(), r)
	}
}
