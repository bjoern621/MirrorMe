package publish

import (
	"time"

	"golang.org/x/sys/windows"

	"bjoernblessin.de/go-utils/util/logger"
)

const (
	exitStallPollDuration  = 250 * time.Millisecond
	exitStallGraceDuration = 2 * time.Second

	// stillActiveExitCode answers GetExitCodeProcess for a process that has not begun to exit.
	stillActiveExitCode = 259
)

// killOnStalledExit ends a child that has begun to exit and stays stuck there.
// Such a child carries its exit code while one thread hangs in teardown and every handle stays open,
// so that cmd.Wait and the readers of its output would wait until something kills it.
// A crashed GStreamer publish child ends this way.
// waited closes once cmd.Wait has returned.
func (c *child) killOnStalledExit(waited <-chan struct{}, tag string) {
	process, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(c.cmd.Process.Pid))
	if err != nil {
		logger.Warnf("%s cannot be watched for a stalled exit: %v", tag, err)
		return
	}
	defer windows.CloseHandle(process)

	exitCode, stalled := awaitStalledExit(func() (uint32, error) {
		var code uint32
		err := windows.GetExitCodeProcess(process, &code)
		return code, err
	}, waited, exitStallPollDuration, exitStallGraceDuration)
	if !stalled {
		return
	}

	logger.Warnf("%s hung while exiting with code %d, killing it", tag, exitCode)
	// A child that ended cleanly has no failure to report, whatever code the kill leaves behind.
	if exitCode == 0 {
		c.stopped.Store(true)
	}
	c.cmd.Process.Kill()
}

// awaitStalledExit polls until the process shows an exit code and then gives it the grace to finish.
// It reports the code and true when that grace runs out, and false once waited closes.
func awaitStalledExit(exitCode func() (uint32, error), waited <-chan struct{}, poll, grace time.Duration) (uint32, bool) {
	ticker := time.NewTicker(poll)
	defer ticker.Stop()

	for {
		select {
		case <-waited:
			return 0, false
		case <-ticker.C:
		}

		code, err := exitCode()
		if err != nil || code == stillActiveExitCode {
			continue
		}

		select {
		case <-waited:
			return 0, false
		case <-time.After(grace):
			return code, true
		}
	}
}
