package publish

import (
	"errors"
	"testing"
	"time"
)

const (
	testExitStallPoll  = time.Millisecond
	testExitStallGrace = 20 * time.Millisecond
	testExitStallLimit = 5 * time.Second
)

func TestAProcessStuckAfterItsExitCodeCountsAsStalled(t *testing.T) {
	code, stalled := awaitStalledExit(func() (uint32, error) { return 2, nil }, make(chan struct{}), testExitStallPoll, testExitStallGrace)

	if !stalled || code != 2 {
		t.Fatalf("got code %d, stalled %v, want code 2 and a stall", code, stalled)
	}
}

func TestARunningProcessNeverCountsAsStalled(t *testing.T) {
	waited := make(chan struct{})
	time.AfterFunc(10*testExitStallGrace, func() { close(waited) })

	if _, stalled := awaitStalledExit(func() (uint32, error) { return stillActiveExitCode, nil }, waited, testExitStallPoll, testExitStallGrace); stalled {
		t.Fatal("a process still running counted as stalled")
	}
}

func TestAnExitThatFinishesWithinTheGraceDoesNotCountAsStalled(t *testing.T) {
	waited := make(chan struct{})
	exitCode := func() (uint32, error) {
		close(waited)
		return 2, nil
	}

	if _, stalled := awaitStalledExit(exitCode, waited, testExitStallPoll, testExitStallLimit); stalled {
		t.Fatal("an exit that finished counted as stalled")
	}
}

func TestAFailedExitCodeQueryIsNoStall(t *testing.T) {
	waited := make(chan struct{})
	time.AfterFunc(10*testExitStallGrace, func() { close(waited) })

	if _, stalled := awaitStalledExit(func() (uint32, error) { return 2, errors.New("access denied") }, waited, testExitStallPoll, testExitStallGrace); stalled {
		t.Fatal("a query that failed counted as a stall")
	}
}
