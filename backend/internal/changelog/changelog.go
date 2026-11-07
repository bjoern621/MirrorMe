// Package changelog reads CHANGELOG.md, the one place that holds a release's notes.
//
// The file holds an Unreleased section on top and one section per release below it, newest first.
// Each section groups its entries under the kinds in Kinds.
// A release's GitHub text derives from this file.
// `task release` picks the next version from it and cuts it (docs/releasing.md).
package changelog

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"bjoernblessin.de/mirrorme/internal/release"
)

// Kinds lists every heading that a section may group its entries under.
// `task release` picks the bump from them (taskfiles/release.sh).
var Kinds = []string{"Added", "Changed", "Fixed", "Removed"}

// unreleased names the section that collects entries between releases.
const unreleased = "Unreleased"

var (
	ErrUnknownKind   = errors.New("entry kind outside changelog.Kinds")
	ErrNoUnreleased  = errors.New("no Unreleased section")
	ErrNotAVersion   = errors.New("no vX.Y.Z version")
	ErrNoSuchVersion = errors.New("no section for this version")
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
