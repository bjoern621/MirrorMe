package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

// sleepSubcommand keeps a run of this binary up until it is killed: the app an applier waits for.
const sleepSubcommand = "test-sleep"

// TestMain lets the test binary stand in for the backend's:
// the launcher and the applier re-execute os.Executable(), which under go test is this one.
func TestMain(m *testing.M) {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case Subcommand:
			os.Exit(Main(os.Args[2:]))
		case LaunchSubcommand:
			os.Exit(Launch(os.Args[2:]))
		case sleepSubcommand:
			time.Sleep(time.Minute)
			os.Exit(0)
		}
	}
	os.Exit(m.Run())
}

// stageSwap lays out an install and a staged release that replaces it,
// and answers the staging directory and the installed binary.
func stageSwap(t *testing.T, dir string) (target string, binary string) {
	t.Helper()
	target = filepath.Join(t.TempDir(), "mirrorme")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("laying out the install: %v", err)
	}
	binary = filepath.Join(target, "mirrorme")
	if err := os.WriteFile(binary, []byte("old"), 0o755); err != nil {
		t.Fatalf("writing the old binary: %v", err)
	}

	archive := filepath.Join(dir, "mirrorme-0.5.1-linux-x86_64-portable.tar.gz")
	writeTarball(t, archive, "mirrorme-0.5.1-linux-x86_64-portable", map[string]string{"mirrorme": "new"})
	pending := Pending{Tag: "v0.5.1", File: archive, Method: MethodSwap, Target: target, Launch: binary}
	if err := WritePending(dir, pending); err != nil {
		t.Fatalf("recording the release: %v", err)
	}
	return target, binary
}

// startApp runs a process standing in for the app, and answers it running.
func startApp(t *testing.T) *exec.Cmd {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locating the test binary: %v", err)
	}
	app := exec.Command(exe, sleepSubcommand)
	if err := app.Start(); err != nil {
		t.Fatalf("starting the stand-in app: %v", err)
	}
	t.Cleanup(func() {
		_ = app.Process.Kill()
		_ = app.Wait()
	})
	return app
}

// endApp kills the stand-in app and reaps it: a zombie still answers a signal.
func endApp(t *testing.T, app *exec.Cmd) {
	t.Helper()
	_ = app.Process.Kill()
	_ = app.Wait()
}

func readBinary(t *testing.T, binary string) string {
	t.Helper()
	got, err := os.ReadFile(binary)
	if err != nil {
		t.Fatalf("reading the installed binary: %v", err)
	}
	return string(got)
}

// waitFor polls until the installed binary holds want, or fails at the deadline.
func waitFor(t *testing.T, binary, want string) {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for {
		if got := readBinary(t, binary); got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the installed binary holds %q at the deadline, want %q", readBinary(t, binary), want)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// The shell ends the backend by killing its whole process tree,
// so an applier left in that tree dies before it installs anything.
// After the install call answers, the applier is under no process that call ran.
func TestTheApplierIsOutsideTheTreeOfTheRunThatInstalls(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("a process's parent is read out of /proc")
	}
	isolateConfig(t)
	t.Setenv(EnvCheck, "1")
	t.Setenv(EnvFlatpak, "")
	dir, err := StageDir()
	if err != nil {
		t.Fatalf("resolving the staging directory: %v", err)
	}
	stageSwap(t, dir)
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locating the test binary: %v", err)
	}

	m := New("0.5.0", Portable, exe, func(State) {})
	if err := m.Install(); err != nil {
		t.Fatalf("installing: %v", err)
	}

	pid, parent := findApplier(t, dir)
	t.Cleanup(func() {
		if p, err := os.FindProcess(pid); err == nil {
			_ = p.Kill()
		}
	})
	if parent == os.Getpid() {
		t.Fatalf("applier %d is a child of the run that installed it, so a kill of that run's tree takes it", pid)
	}
}

// findApplier answers the running applier for dir and its parent, out of /proc.
func findApplier(t *testing.T, dir string) (pid, parent int) {
	t.Helper()
	entries, err := os.ReadDir("/proc")
	if err != nil {
		t.Fatalf("reading /proc: %v", err)
	}
	for _, e := range entries {
		candidate, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join("/proc", e.Name(), "cmdline"))
		if err != nil {
			continue
		}
		args := strings.Split(string(cmdline), "\x00")
		if len(args) < 3 || args[1] != Subcommand || args[2] != StageFlag+dir {
			continue
		}
		stat, err := os.ReadFile(filepath.Join("/proc", e.Name(), "stat"))
		if err != nil {
			t.Fatalf("reading the applier's stat: %v", err)
		}
		// "pid (comm) state ppid ...", comm free to hold spaces.
		fields := strings.Fields(string(stat)[strings.LastIndex(string(stat), ")")+1:])
		parent, err = strconv.Atoi(fields[1])
		if err != nil {
			t.Fatalf("reading the applier's parent: %v", err)
		}
		return candidate, parent
	}
	t.Fatalf("no applier is running for %s", dir)
	return 0, 0
}

// The applier installs nothing while the app is up, and installs once it is gone.
// Its own binary stays in the staging directory afterwards, Windows keeping a running one,
// and the record and the release go.
func TestTheApplierWaitsForTheAppAndThenInstalls(t *testing.T) {
	isolateConfig(t)
	dir := t.TempDir()
	target, binary := stageSwap(t, dir)
	applier, err := applierCopy(dir)
	if err != nil {
		t.Fatalf("copying the applier: %v", err)
	}
	app := startApp(t)

	run := exec.Command(applier, Subcommand, StageFlag+dir, WaitFlag+strconv.Itoa(app.Process.Pid))
	run.Stderr = os.Stderr
	if err := run.Start(); err != nil {
		t.Fatalf("starting the applier: %v", err)
	}

	time.Sleep(time.Second)
	if got := readBinary(t, binary); got != "old" {
		t.Fatalf("the installed binary holds %q while the app is up, want %q", got, "old")
	}

	endApp(t, app)
	waitFor(t, binary, "new")
	if err := run.Wait(); err != nil {
		t.Fatalf("the applier ended with %v", err)
	}
	if got := names(t, dir); len(got) != 1 || got[0] != filepath.Base(applier) {
		t.Fatalf("the staging directory holds %v after the install, want the applier alone", got)
	}
	if _, err := os.Stat(outgoingOf(target)); !os.IsNotExist(err) {
		t.Fatalf("the old tree stands after the applier ended, stat answers %v", err)
	}
}
