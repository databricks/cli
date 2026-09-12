package fuse_test

import (
	"os"
	"runtime"
	"testing"

	"github.com/databricks/cli/experimental/ssh/internal/fuse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const processStat = "4242 (databricks) S 1 1234 1234 0 -1 4194560 5407 0 0 0 33 12 0 0 20 0 14 0 8613244 1234"

var registration = fuse.Registration{PID: 4242, PIDNamespaceID: 4026531836, StartTime: 8613244}

func TestParseRegistration(t *testing.T) {
	for _, tc := range []struct {
		name, namespace, stat, err string
	}{
		{name: "valid", namespace: "pid:[4026531836]", stat: processStat},
		{name: "spaces and parentheses", namespace: "pid:[4026531836]", stat: "4242 (weird ) name)) S 1 1234 1234 0 -1 4194560 5407 0 0 0 33 12 0 0 20 0 14 0 8613244"},
		{name: "wrong namespace", namespace: "mnt:[4026531836]", stat: processStat, err: "unexpected PID namespace"},
		{name: "incomplete namespace", namespace: "pid:[4026531836", stat: processStat, err: "unexpected PID namespace"},
		{name: "empty namespace", namespace: "pid:[]", stat: processStat, err: "failed to parse the PID namespace"},
		{name: "namespace overflow", namespace: "pid:[4294967296]", stat: processStat, err: "failed to parse the PID namespace"},
		{name: "missing name", namespace: "pid:[4026531836]", stat: "4242 databricks S 1", err: "missing process name"},
		{name: "truncated stat", namespace: "pid:[4026531836]", stat: "4242 (databricks) S 1", err: "missing start time"},
		{name: "invalid time", namespace: "pid:[4026531836]", stat: "4242 (databricks) S 1 1234 1234 0 -1 4194560 5407 0 0 0 33 12 0 0 20 0 14 0 invalid", err: "failed to parse process start time"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fuse.ParseRegistration(registration.PID, tc.namespace, tc.stat)
			if tc.err != "" {
				require.ErrorContains(t, err, tc.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, registration, got)
		})
	}
}

func TestSelf(t *testing.T) {
	r, err := fuse.Self()
	if runtime.GOOS != "linux" {
		require.Error(t, err)
		return
	}
	require.NoError(t, err)
	assert.Equal(t, os.Getpid(), r.PID)
	assert.Positive(t, r.PIDNamespaceID)
	assert.Positive(t, r.StartTime)
}
