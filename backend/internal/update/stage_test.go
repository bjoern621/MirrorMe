package update

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// isolateConfig points os.UserConfigDir at a fresh temp directory,
// so a staging directory a test resolves lands in one of the test's own.
// All three variables, os.UserConfigDir reading a different one per platform.
func isolateConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("AppData", dir)
	t.Setenv("HOME", dir)

	got, err := os.UserConfigDir()
	if err != nil {
		t.Fatalf("isolating the config directory: %v", err)
	}
	if !strings.HasPrefix(got, dir) {
		t.Fatalf("os.UserConfigDir is %s, outside the temp directory %s: this test would write into the real config directory", got, dir)
	}
	return dir
}

// stageRelease writes a release into dir as a finished download leaves it,
// and answers the pending record it wrote.
func stageRelease(t *testing.T, dir, tag string) Pending {
	t.Helper()
	file := filepath.Join(dir, "mirrorme-"+strings.TrimPrefix(tag, "v")+"-linux-x86_64-portable.tar.gz")
	if err := os.WriteFile(file, []byte("release"), 0o600); err != nil {
		t.Fatalf("writing the release: %v", err)
	}
	pending := Pending{Tag: tag, File: file, Method: MethodSwap, Target: t.TempDir(), Launch: "mirrorme"}
	if err := WritePending(dir, pending); err != nil {
		t.Fatalf("recording the release: %v", err)
	}
	return pending
}

func names(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	var got []string
	for _, e := range entries {
		got = append(got, e.Name())
	}
	return got
}

// An entry the clear cannot remove leaves the rest of the directory cleared all the same:
// a release nobody installs is disk somebody has to reclaim, whatever else is in the way.
func TestClearingRemovesWhatItCanPastAnEntryItCannot(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("a directory without write permission holds its entries for an ordinary user on a POSIX filesystem")
	}
	dir := t.TempDir()

	// Sorts first, so a clear that stops at its first failure leaves everything after it.
	held := filepath.Join(dir, "a-held")
	if err := os.MkdirAll(held, 0o755); err != nil {
		t.Fatalf("laying out the held directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(held, "x"), []byte("x"), 0o600); err != nil {
		t.Fatalf("writing into the held directory: %v", err)
	}
	if err := os.Chmod(held, 0o500); err != nil {
		t.Fatalf("holding the directory: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(held, 0o755) })
	stageRelease(t, dir, "v0.5.0")

	if err := ClearStage(dir); err == nil {
		t.Fatalf("clearing past %s reports no error", held)
	}

	if got := names(t, dir); len(got) != 1 || got[0] != "a-held" {
		t.Fatalf("the directory holds %v after the clear, with only %s in the way", got, held)
	}
}

// A release staged behind the build now running is the one this run is,
// left by the applier that installed it, and it is swept on the way up
// rather than offered for a restart that would install what already runs.
func TestAStagedReleaseBehindTheRunIsSweptOnStart(t *testing.T) {
	isolateConfig(t)
	t.Setenv(EnvCheck, "1")
	t.Setenv(EnvFlatpak, "")
	dir, err := StageDir()
	if err != nil {
		t.Fatalf("resolving the staging directory: %v", err)
	}
	stageRelease(t, dir, "v0.5.0")
	// What a finished applier leaves behind on Windows, which keeps a running binary.
	if err := os.WriteFile(filepath.Join(dir, "applier.exe"), []byte("x"), 0o700); err != nil {
		t.Fatalf("writing the leftover applier: %v", err)
	}

	m := New("0.5.0", Portable, filepath.Join(t.TempDir(), "mirrorme-backend"), func(State) {})

	if got := m.State().Stage; got != StageUnchecked {
		t.Fatalf("the install starts at stage %d with the running release staged, want %d", got, StageUnchecked)
	}
	if got := names(t, dir); len(got) != 0 {
		t.Fatalf("the staging directory holds %v after the start", got)
	}
}

// A release staged ahead of the running build is still the one a restart installs.
func TestAStagedReleaseAheadOfTheRunSurvivesStart(t *testing.T) {
	isolateConfig(t)
	t.Setenv(EnvCheck, "1")
	t.Setenv(EnvFlatpak, "")
	dir, err := StageDir()
	if err != nil {
		t.Fatalf("resolving the staging directory: %v", err)
	}
	pending := stageRelease(t, dir, "v0.6.0")

	m := New("0.5.0", Portable, filepath.Join(t.TempDir(), "mirrorme-backend"), func(State) {})

	state := m.State()
	if state.Stage != StageReady || state.Latest != pending.Tag {
		t.Fatalf("the install starts at stage %d naming %q, want %d naming %q", state.Stage, state.Latest, StageReady, pending.Tag)
	}
	if _, err := os.Stat(pending.File); err != nil {
		t.Fatalf("the staged release is gone after the start: %v", err)
	}
}

// A directory with no record in it holds leftovers and nothing else, and the start reclaims them.
func TestLeftoversWithNothingStagedAreSweptOnStart(t *testing.T) {
	isolateConfig(t)
	t.Setenv(EnvCheck, "1")
	t.Setenv(EnvFlatpak, "")
	dir, err := StageDir()
	if err != nil {
		t.Fatalf("resolving the staging directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mirrorme-0.6.0-linux-x86_64-portable.tar.gz.partial"), []byte("x"), 0o600); err != nil {
		t.Fatalf("writing the partial download: %v", err)
	}

	New("0.5.0", Portable, filepath.Join(t.TempDir(), "mirrorme-backend"), func(State) {})

	if got := names(t, dir); len(got) != 0 {
		t.Fatalf("the staging directory holds %v after the start", got)
	}
}
