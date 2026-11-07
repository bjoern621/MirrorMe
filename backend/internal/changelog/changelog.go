// Package changelog reads and cuts CHANGELOG.md, the one place that holds a release's notes.
//
// The file holds an Unreleased section on top and one section per release below it, newest first.
// Each section groups its entries under the kinds in Kinds.
// A release's GitHub text, the next version's number and the cut that a release commits
// all derive from this file (docs/releasing.md).
package changelog

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"bjoernblessin.de/go-utils/util/assert"

	"bjoernblessin.de/screenshare/internal/release"
)

// Kinds lists every heading that a section may group its entries under.
// The bump that each kind asks for sits in Next.
var Kinds = []string{"Added", "Changed", "Fixed", "Removed"}

// unreleased names the section that collects entries between releases.
const unreleased = "Unreleased"

var (
	ErrUnknownKind       = errors.New("entry kind outside changelog.Kinds")
	ErrNoUnreleased      = errors.New("no Unreleased section")
	ErrNothingUnreleased = errors.New("no entries under Unreleased")
	ErrAlreadyCut        = errors.New("version cut already, with more entries under Unreleased")
	ErrNotNewer          = errors.New("version not past the newest one in the changelog")
	ErrNotAVersion       = errors.New("no vX.Y.Z version")
	ErrNoSuchVersion     = errors.New("no section for this version")
)

// Log holds a parsed changelog.
// Printing it gives back the text it was parsed from, byte for byte.
type Log struct {
	preamble string    // Everything above the first "## " heading.
	sections []section // In file order: Unreleased, then releases newest first.
}

type section struct {
	heading string // The heading line without its "## ": "v0.12.1 (2026-11-05)".
	version string // First word of the heading. Empty for Unreleased.
	body    string // Every line up to the next heading, blank lines included.
}

// Parse reads a changelog.
// Errors if a heading names neither Unreleased nor a version, if a kind falls outside Kinds,
// or if no Unreleased section exists.
func Parse(text string) (Log, error) {
	var log Log
	lines := strings.SplitAfter(text, "\n")
	current := -1
	for _, line := range lines {
		heading, found := strings.CutPrefix(line, "## ")
		if !found {
			if current < 0 {
				log.preamble += line
			} else {
				log.sections[current].body += line
			}
			continue
		}

		s, err := parseHeading(strings.TrimSuffix(heading, "\n"))
		if err != nil {
			return Log{}, err
		}
		log.sections = append(log.sections, s)
		current++
	}

	if len(log.sections) == 0 || log.sections[0].version != "" {
		return Log{}, ErrNoUnreleased
	}
	for _, s := range log.sections {
		if _, err := kinds(s.body); err != nil {
			return Log{}, fmt.Errorf("under %q: %w", s.heading, err)
		}
	}
	return log, nil
}

func parseHeading(heading string) (section, error) {
	if heading == unreleased {
		return section{heading: heading}, nil
	}
	version, _, _ := strings.Cut(heading, " ")
	if _, ok := release.Parse(version); !ok || !strings.HasPrefix(version, "v") {
		return section{}, fmt.Errorf("heading %q: %w", heading, ErrNotAVersion)
	}
	return section{heading: heading, version: version}, nil
}

// kinds answers the kinds in a section's body that hold at least one line of text.
func kinds(body string) ([]string, error) {
	var found []string
	kind := ""
	for line := range strings.SplitSeq(body, "\n") {
		if name, ok := strings.CutPrefix(line, "### "); ok {
			kind = strings.TrimSpace(name)
			if !slices.Contains(Kinds, kind) {
				return nil, fmt.Errorf("%w: %q", ErrUnknownKind, kind)
			}
			continue
		}
		if strings.TrimSpace(line) != "" && kind != "" && !slices.Contains(found, kind) {
			found = append(found, kind)
		}
	}
	return found, nil
}

func (l Log) String() string {
	var b strings.Builder
	b.WriteString(l.preamble)
	for _, s := range l.sections {
		b.WriteString("## " + s.heading + "\n" + s.body)
	}
	return b.String()
}

// Versions answers every released version that the log holds, newest first.
func (l Log) Versions() []string {
	var out []string
	for _, s := range l.sections[1:] {
		out = append(out, s.version)
	}
	return out
}

// Next answers the version that a release of the Unreleased entries takes, counting up from latest.
// A fix alone takes a patch, and any other kind a minor.
// A removal takes a major from 1.0.0 on.
// Errors if Unreleased holds no entries or latest carries no version.
func (l Log) Next(latest string) (string, error) {
	pending, err := kinds(l.sections[0].body)
	assert.Assert(err == nil, "Parse checks every section's kinds", err)
	if len(pending) == 0 {
		return "", ErrNothingUnreleased
	}
	v, ok := release.Parse(latest)
	if !ok {
		return "", fmt.Errorf("latest release %q: %w", latest, ErrNotAVersion)
	}

	switch {
	case slices.Contains(pending, "Removed") && v[0] >= 1:
		v = [3]int{v[0] + 1, 0, 0}
	case slices.ContainsFunc(pending, func(k string) bool { return k != "Fixed" }):
		v = [3]int{v[0], v[1] + 1, 0}
	default:
		v[2]++
	}
	return fmt.Sprintf("v%d.%d.%d", v[0], v[1], v[2]), nil
}

// Cut moves the Unreleased entries under a new section for version, dated date (YYYY-MM-DD).
// A version cut already with nothing left under Unreleased answers the log unchanged,
// so a release interrupted after its cut can run again.
// Errors if version is not vX.Y.Z or not past the newest release in the log,
// if Unreleased is empty, or if version is cut already and Unreleased holds more entries.
func (l Log) Cut(version, date string) (Log, error) {
	_, err := time.Parse(time.DateOnly, date)
	assert.Assert(err == nil, "a cut is dated YYYY-MM-DD", date)

	if _, ok := release.Parse(version); !ok || !strings.HasPrefix(version, "v") {
		return Log{}, fmt.Errorf("%q: %w", version, ErrNotAVersion)
	}
	pending, err := kinds(l.sections[0].body)
	assert.Assert(err == nil, "Parse checks every section's kinds", err)

	if slices.Contains(l.Versions(), version) {
		if len(pending) > 0 {
			return Log{}, fmt.Errorf("%s: %w", version, ErrAlreadyCut)
		}
		return l, nil
	}
	if len(pending) == 0 {
		return Log{}, ErrNothingUnreleased
	}
	if newest := l.Versions(); len(newest) > 0 && release.Compare(newest[0], version) != release.StateBehind {
		return Log{}, fmt.Errorf("%s against %s: %w", version, newest[0], ErrNotNewer)
	}

	released := section{heading: version + " (" + date + ")", version: version, body: l.sections[0].body}
	sections := []section{{heading: unreleased, body: "\n"}, released}
	return Log{preamble: l.preamble, sections: append(sections, l.sections[1:]...)}, nil
}

// Entries answers the body of version's section, without the blank lines around it.
// Errors if the log holds no section for version.
func (l Log) Entries(version string) (string, error) {
	for _, s := range l.sections[1:] {
		if s.version == version {
			return strings.TrimSpace(s.body), nil
		}
	}
	return "", fmt.Errorf("%s: %w", version, ErrNoSuchVersion)
}

// Previous answers the newest of tags older than version, or "" where none is.
// A tag carrying no version is passed over.
func Previous(tags []string, version string) string {
	previous := ""
	for _, tag := range tags {
		if !release.Comparable(tag) || release.Compare(tag, version) != release.StateBehind {
			continue
		}
		if previous == "" || release.Compare(previous, tag) == release.StateBehind {
			previous = tag
		}
	}
	return previous
}

// Newest answers the highest of tags, or "" where none carries a version.
func Newest(tags []string) string {
	newest := ""
	for _, tag := range tags {
		if !release.Comparable(tag) {
			continue
		}
		if newest == "" || release.Compare(newest, tag) == release.StateBehind {
			newest = tag
		}
	}
	return newest
}
