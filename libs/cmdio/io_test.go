package cmdio

import (
	"testing"
	"time"
)

func TestAcquireTeaProgramWaitsForRelease(t *testing.T) {
	c := &cmdIO{}
	// acquireTeaProgram only stores the pointer, so a nil program suffices.
	c.acquireTeaProgram(nil)

	acquired := make(chan struct{})
	go func() {
		c.acquireTeaProgram(nil)
		close(acquired)
	}()

	select {
	case <-acquired:
		t.Fatal("second acquireTeaProgram returned while the first program was still active")
	case <-time.After(50 * time.Millisecond):
	}

	// Release on a goroutine so a regressed deadlock fails the timeout below
	// instead of hanging the test.
	go c.releaseTeaProgram()

	select {
	case <-acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("second acquireTeaProgram did not return after the first program was released")
	}

	c.releaseTeaProgram()
}
