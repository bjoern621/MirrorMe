// Package qa holds what a test run on one machine records about one release,
// and renders the records into the matrix of releases against machines (docs/releasing.md).
//
// One record per release and machine, at Record.Path under qa/.
// Every page showing records derives from the files on each render.
package qa

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"bjoernblessin.de/go-utils/util/assert"
)

// Record holds one run of the hardware suite on one machine against one release.
type Record struct {
	Version string  `json:"version"` // The release tag that the run tested: "v0.12.1".
	Date    string  `json:"date"`    // YYYY-MM-DD of the run.
	Machine Machine `json:"machine"`
	Tests   Tests   `json:"tests"`
}

// Machine describes the hardware and the media stack that the run found.
// A field the machine did not answer stays empty.
type Machine struct {
	Name      string `json:"name"`   // Chosen by whoever records. Must be unique per machine.
	OS        string `json:"os"`     // "NixOS 26.05 (Yarara)".
	Kernel    string `json:"kernel"` // Empty on Windows, whose OS field carries the build.
	Arch      string `json:"arch"`   // GOARCH: "amd64", "arm64".
	CPU       string `json:"cpu"`
	GPUs      []GPU  `json:"gpus"`
	VADriver  string `json:"va_driver"` // Whole string that the VA driver introduces itself with.
	GStreamer string `json:"gstreamer"`
	FFmpeg    string `json:"ffmpeg"`
}

// GPU describes one display adapter.
type GPU struct {
	Name          string `json:"name"`
	Driver        string `json:"driver"`         // Kernel module on Linux: "amdgpu", "nvidia".
	DriverVersion string `json:"driver_version"` // Empty where the driver reports none.
}

// Tests counts what `go test -tags hardware` answered, per leaf test.
type Tests struct {
	Passed   int      `json:"passed"`
	Failed   int      `json:"failed"`
	Skipped  int      `json:"skipped"`
	Failures []string `json:"failures"` // "ffmpeg.TestArgs/nvenc", sorted.
	Skips    []Skip   `json:"skips"`    // Sorted by test.
}

// Skip names a test that the machine could not run and the reason that the test gave.
type Skip struct {
	Test   string `json:"test"`
	Reason string `json:"reason"`
}

var notSlug = regexp.MustCompile(`[^a-z0-9]+`)

// Path is where the record sits, relative to qa/: "runs/v0.12.1/desk.json".
// The machine name is folded to lowercase letters, digits and hyphens.
func (r Record) Path() string {
	assert.Assert(r.Version != "" && r.Machine.Name != "", "a record names its version and machine", r.Version, r.Machine.Name)

	slug := strings.Trim(notSlug.ReplaceAllString(strings.ToLower(r.Machine.Name), "-"), "-")
	assert.Assert(slug != "", "a machine name carries at least one letter or digit", r.Machine.Name)
	return "runs/" + r.Version + "/" + slug + ".json"
}

// Write puts the record at its Path under dir, replacing one the machine recorded before.
func (r Record) Write(dir string) error {
	path := filepath.Join(dir, filepath.FromSlash(r.Path()))
	data, err := json.MarshalIndent(r, "", "  ")
	assert.Assert(err == nil, "a record always marshals", err)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// Load reads every record under dir/runs.
// A directory without runs answers none.
// Errors if a file fails to parse or sits at a path that its own contents disagree with.
func Load(dir string) ([]Record, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "runs", "*", "*.json"))
	assert.Assert(err == nil, "the pattern is well-formed", err)

	var records []Record
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		var r Record
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", path, err)
		}
		if r.Version == "" || r.Machine.Name == "" {
			return nil, fmt.Errorf("%s names no version or no machine", path)
		}
		if rel, _ := filepath.Rel(dir, path); filepath.ToSlash(rel) != r.Path() {
			return nil, fmt.Errorf("%s holds the record for %s", path, r.Path())
		}
		records = append(records, r)
	}
	return records, nil
}
