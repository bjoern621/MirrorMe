package qa

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Event decodes one line of `go test -json`.
type Event struct {
	Action  string
	Package string
	Test    string
	Output  string
	Elapsed float64 // Seconds, on a pass, fail or skip.
}

// fileLine matches the prefix that testing puts ahead of a t.Skip message: "    gpu_test.go:53: ".
var fileLine = regexp.MustCompile(`^\s*\S+\.go:\d+: `)

type outcome struct {
	action string
	reason string // Last line that the test logged, for a skip.
}

// Summarize counts the leaf tests in a `go test -json` stream.
// A test with subtests counts through its subtests alone.
// A package that failed with no failing test (a build error, a panic in TestMain)
// counts as one failure under the package's name.
// progress, where set, sees every event as it is read.
// Errors if the stream holds no event or a line that is no event.
func Summarize(r io.Reader, progress func(Event)) (Tests, error) {
	tests := map[string]*outcome{} // Keyed by "<package>.<test>".
	failedPackages := []string{}
	read := 0

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for scanner.Scan() {
		var e Event
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return Tests{}, fmt.Errorf("reading a test event %q: %w", scanner.Text(), err)
		}
		read++
		if progress != nil {
			progress(e)
		}

		pkg := path.Base(e.Package)
		if e.Test == "" {
			if e.Action == "fail" {
				failedPackages = append(failedPackages, pkg)
			}
			continue
		}
		key := pkg + "." + e.Test
		o := tests[key]
		if o == nil {
			o = &outcome{}
			tests[key] = o
		}
		switch e.Action {
		case "pass", "fail", "skip":
			o.action = e.Action
		case "output":
			line := strings.TrimSpace(e.Output)
			if line != "" && !strings.HasPrefix(line, "=== ") && !strings.HasPrefix(line, "--- ") {
				o.reason = strings.TrimSpace(fileLine.ReplaceAllString(e.Output, ""))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return Tests{}, fmt.Errorf("reading test events: %w", err)
	}
	if read == 0 {
		return Tests{}, errors.New("the test run printed no events")
	}

	return count(tests, failedPackages), nil
}

func count(tests map[string]*outcome, failedPackages []string) Tests {
	var t Tests
	failingPackages := map[string]bool{}
	for key, o := range tests {
		if isParent(tests, key) {
			continue
		}
		switch o.action {
		case "pass":
			t.Passed++
		case "fail":
			t.Failed++
			t.Failures = append(t.Failures, key)
			pkg, _, _ := strings.Cut(key, ".")
			failingPackages[pkg] = true
		case "skip":
			t.Skipped++
			t.Skips = append(t.Skips, Skip{Test: key, Reason: o.reason})
		}
	}
	for _, pkg := range failedPackages {
		if !failingPackages[pkg] {
			t.Failed++
			t.Failures = append(t.Failures, pkg+" (failed outside a test)")
		}
	}

	slices.Sort(t.Failures)
	slices.SortFunc(t.Skips, func(a, b Skip) int { return strings.Compare(a.Test, b.Test) })
	return t
}

func isParent(tests map[string]*outcome, key string) bool {
	for other := range tests {
		if strings.HasPrefix(other, key+"/") {
			return true
		}
	}
	return false
}
