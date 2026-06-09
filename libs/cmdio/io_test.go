package cmdio

import (
	"testing"
	"time"
)

func TestAcquireTeaProgramWaitsForRelease(t *testing.T) {
	c := &cmdIO{}
	// acquireTeaProgram only stores the pointer, so nil is enough to exercise
	// the queueing without running a real tea.Program.
	c.acquireTeaProgram(nil)

	acquired := make(chan struct{})
	go func() {
		c.acquireTeaProgram(nil)
		close(acquired)
	}()

	// The second acquire must queue behind the active program.
	select {
	case <-acquired:
		t.Fatal("second acquireTeaProgram returned while the first program was still active")
	case <-time.After(50 * time.Millisecond):
	}

	// Release on a separate goroutine: the original implementation deadlocked
	// here because the waiter held teaMu while blocked on teaDone, which
	// releaseTeaProgram needs to close it.
	go c.releaseTeaProgram()

	select {
	case <-acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("second acquireTeaProgram did not return after the first program was released")
	}

	c.releaseTeaProgram()
}
