package update

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"bjoernblessin.de/go-utils/util/logger"
)

// The applier: the one run that replaces the app's own files.
//
// A process of its own, from a copy of this binary outside the tree it is replacing,
// started by the run that is about to exit.
// Nothing it touches is open by anything: it waits for the app to be gone,
// puts the files in place, starts the app again and deletes what it worked from.
//
// It logs to a run log of its own, beside the backend's (cmd/backend, openLog):
// the app it was started by is gone, and neither half has a console on an install.

// Subcommand runs the applier instead of the app, on the executable both share.
const Subcommand = "update-apply"

// LaunchSubcommand runs Launch instead of the app, on the same executable.
const LaunchSubcommand = "update-launch"

// StageFlag names the staging directory, WaitFlag the process to outlive.
const (
	StageFlag = "--stage="
	WaitFlag  = "--wait-pid="
)

// waitLimit bounds how long the applier waits for the app to close.
// A window asking to save on the way out is what the wait is for,
// and past this the install is abandoned with the running app untouched.
const waitLimit = 60 * time.Second

// waitStep is how often the wait looks.
const waitStep = 200 * time.Millisecond

// Launch starts the applier and exits as soon as it is up.
//
// The shell ends the backend by killing its whole process tree,
// and every process whose parent is still in that tree goes with it.
// The applier's parent is this run, gone before the backend answers the install,
// so the kill stops short of it.
//
// args are the applier's own, passed through unread.
// A failure reaches the backend on stderr, which is what it reports.
func Launch(args []string) int {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "locating the applier: %v\n", err)
		return 1
	}

	command := exec.Command(exe, append([]string{Subcommand}, args...)...)
	detach(command)
	if err := command.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "starting the applier: %v\n", err)
		return 1
	}
	if err := command.Process.Release(); err != nil {
		fmt.Fprintf(os.Stderr, "releasing the applier: %v\n", err)
	}
	return 0
}

// Main runs the applier and answers the process's exit status.
//
// Every failure leaves the install exactly as it was:
// the staged files stay where they are, and the next start finds the release still staged.
// A failed install starts the app again, since the reader closed it for this and nothing else will.
func Main(args []string) int {
	stage, pid, err := applierArgs(args)
	if err != nil {
		logger.Warnf("%v", err)
		return 1
	}

	// This run inherits the app's working directory, usually the install itself,
	// and Windows refuses to rename a directory that a process stands in.
	if err := os.Chdir(os.TempDir()); err != nil {
		logger.Warnf("leaving the install directory: %v", err)
	}

	pending, staged := ReadPending(stage)
	if !staged {
		logger.Warnf("nothing is staged in %s", stage)
		return 1
	}

	if pid > 0 && !closed(pid) {
		logger.Warnf("process %d is still running after %s, so nothing was installed", pid, waitLimit)
		return 1
	}

	if err := install(pending); err != nil {
		logger.Warnf("installing %s: %v", pending.Tag, err)
		if err := relaunch(pending.Launch); err != nil {
			logger.Warnf("starting %s: %v", pending.Launch, err)
		}
		return 1
	}
	logger.Infof("release %s is installed in %s", pending.Tag, pending.Target)

	if err := relaunch(pending.Launch); err != nil {
		// The files are in place, so the install landed and only the start did not.
		// Reported rather than undone: the app the reader opens next is the new one.
		logger.Warnf("starting %s: %v", pending.Launch, err)
	}

	// This run's own binary stays, Windows keeping a running one, and the next start sweeps it.
	if err := ClearStage(stage); err != nil {
		logger.Warnf("clearing %s: %v", stage, err)
	}

	// Last, to keep a tree that a scanner holds from delaying the new app or the clear.
	if pending.Method == MethodSwap {
		outgoing := outgoingOf(pending.Target)
		if err := removeRetrying(outgoing, outgoingWait); err != nil {
			logger.Warnf("removing the old tree at %s, a later start tries again: %v", outgoing, err)
		}
	}
	return 0
}

// applierArgs reads the two flags the applier takes.
func applierArgs(args []string) (stage string, pid int, err error) {
	for _, arg := range args {
		if dir, found := strings.CutPrefix(arg, StageFlag); found {
			stage = dir
			continue
		}
		if raw, found := strings.CutPrefix(arg, WaitFlag); found {
			pid, err = strconv.Atoi(raw)
			if err != nil {
				return "", 0, fmt.Errorf("%s takes a process id: %w", WaitFlag, err)
			}
		}
	}
	if stage == "" {
		return "", 0, fmt.Errorf("%s names the staging directory", StageFlag)
	}
	return stage, pid, nil
}

// closed waits out the app, and answers false where it is still there at the limit.
func closed(pid int) bool {
	deadline := time.Now().Add(waitLimit)
	for time.Now().Before(deadline) {
		if !alive(pid) {
			// The other half of the app closes on its own once this one is gone,
			// and on Windows a file it still holds open is a file that cannot be replaced.
			time.Sleep(2 * time.Second)
			return true
		}
		time.Sleep(waitStep)
	}
	return false
}

// install puts the staged release in place, by the method its channel declared.
func install(pending Pending) error {
	switch pending.Method {
	case MethodRun:
		return run(pending.File)
	case MethodSwap:
		return Swap(pending.File, pending.Target)
	default:
		return fmt.Errorf("a staged release names no way to install it")
	}
}

// run executes the downloaded installer and waits for it.
//
// Silent, because the reader already agreed to this in the app,
// and without a restart of its own: starting the app again is this applier's last step.
func run(path string) error {
	command := exec.Command(path, "/VERYSILENT", "/NORESTART", "/SUPPRESSMSGBOXES", "/NOCANCEL")
	command.Stdout, command.Stderr = os.Stderr, os.Stderr

	if err := command.Run(); err != nil {
		return err
	}
	return nil
}

// relaunch starts the installed app and leaves it running.
func relaunch(launch string) error {
	if _, err := os.Stat(launch); err != nil {
		return err
	}

	command := exec.Command(launch)
	command.Dir = filepath.Dir(launch)
	detach(command)

	if err := command.Start(); err != nil {
		return err
	}
	return command.Process.Release()
}
