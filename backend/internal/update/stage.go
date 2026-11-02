package update

import (
	"encoding/json"
	"os"
	"path/filepath"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
	"bjoernblessin.de/screenshare/internal/text"

	"bjoernblessin.de/go-utils/util/assert"
)

// Where a downloaded release waits, and what says it is waiting there.
//
// One directory per install, beside the settings and the logs,
// so a release that arrived survives the run that fetched it and is found by the next one.
// One tag at a time: staging a release clears whatever an earlier check left,
// a build nobody is going to install being disk somebody has to reclaim.

// configDirName is the app's directory inside os.UserConfigDir, as every package here spells it.
const configDirName = "mirrorme"

// stageDirName is the staging directory inside it.
const stageDirName = "update"

// pendingFileName records the staged release, beside the file it names.
const pendingFileName = "pending.json"

// dirMode keeps the staged files to their owner, as the settings store keeps its own.
const dirMode = 0o700

// Pending is a release on disk, waiting for a restart.
//
// Written whole when a download verifies and read back by whatever installs it,
// including the run after the one that fetched it.
type Pending struct {
	// Tag the release publishes under: "v0.5.1".
	Tag string `json:"tag"`
	// File is the downloaded artifact, an absolute path inside the staging directory.
	File string `json:"file"`
	// Method is how that file is put in place.
	Method Method `json:"method"`
	// Target is the directory holding the running app, which MethodSwap replaces.
	Target string `json:"target"`
	// Launch is the app binary started once the install lands.
	Launch string `json:"launch"`
}

// StageDir is where downloads land, created where it is absent.
//
// A directory that cannot be resolved or created is an Umgebungsfehler and leaves as an error,
// never a path written into anyway.
func StageDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", text.Because(err, screensharev1.TextCode_TEXT_CODE_CONFIG_DIR_UNKNOWN)
	}

	dir := filepath.Join(base, configDirName, stageDirName)
	if err := os.MkdirAll(dir, dirMode); err != nil {
		return "", text.Because(err, screensharev1.TextCode_TEXT_CODE_FILE_UNWRITABLE, text.Path(dir))
	}
	return dir, nil
}

// ReadPending answers the staged release, and false where nothing is staged.
//
// A record naming a file that is gone reads as nothing staged:
// what the record is for is finding the file, so a record without one states nothing.
func ReadPending(dir string) (Pending, bool) {
	assert.Assert(dir != "", "a staged release is read out of a directory")

	raw, err := os.ReadFile(filepath.Join(dir, pendingFileName))
	if err != nil {
		return Pending{}, false
	}

	var pending Pending
	if err := json.Unmarshal(raw, &pending); err != nil {
		return Pending{}, false
	}
	if pending.Tag == "" || pending.File == "" {
		return Pending{}, false
	}
	if _, err := os.Stat(pending.File); err != nil {
		return Pending{}, false
	}
	return pending, true
}

// WritePending records the staged release, replacing whatever stood.
func WritePending(dir string, pending Pending) error {
	assert.Assert(dir != "", "a staged release is recorded in a directory")
	assert.Assert(pending.Tag != "", "a staged release names its tag")
	assert.Assert(pending.File != "", "a staged release names the file it arrived as")
	assert.Assert(pending.Method != MethodNone, "a staged release names how it is installed", pending.Tag)

	raw, err := json.Marshal(pending)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, pendingFileName), raw, 0o600)
}

// ClearStage empties the staging directory, keeping the directory itself
// and the binary this process runs as.
//
// Run before a download, after an install, and on a start that finds nothing worth keeping:
// the first stops two releases sharing the directory,
// the second stops the next run installing the one it is already running,
// and the third reclaims what either left.
// The running binary is the applier clearing what it worked from,
// which Windows keeps until it exits and the next start sweeps.
// An entry that will not go is reported once the rest have gone, so one leftover holds no other.
func ClearStage(dir string) error {
	assert.Assert(dir != "", "a staging directory is cleared by path")

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	var self os.FileInfo
	if exe, err := os.Executable(); err == nil {
		self, _ = os.Stat(exe)
	}

	var failed error
	for _, e := range entries {
		path := filepath.Join(dir, e.Name())
		if self != nil {
			if info, err := os.Stat(path); err == nil && os.SameFile(info, self) {
				continue
			}
		}
		if err := os.RemoveAll(path); err != nil && failed == nil {
			failed = err
		}
	}
	return failed
}
