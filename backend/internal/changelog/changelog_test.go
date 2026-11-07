package changelog

import (
	"errors"
	"strings"
	"testing"
)

const pending = `# Changelog

What changes for somebody taking a release.

## Unreleased

### Fixed

- A slow upload stalls the preview no more.

### Added

- Insights name the dropped frames.

## v0.12.1 (2026-11-05)

### Fixed

- SRT holds its watch window once.
`

const cut = `# Changelog

What changes for somebody taking a release.

## Unreleased

## v0.13.0 (2026-11-07)

### Fixed

- A slow upload stalls the preview no more.

### Added

- Insights name the dropped frames.

## v0.12.1 (2026-11-05)

### Fixed

- SRT holds its watch window once.
`

func mustParse(t *testing.T, text string) Log {
	t.Helper()
	log, err := Parse(text)
	if err != nil {
		t.Fatalf("parsing: %v", err)
	}
	return log
}

func TestParseRoundTrips(t *testing.T) {
	for _, text := range []string{pending, cut} {
		if got := mustParse(t, text).String(); got != text {
			t.Errorf("a parsed log prints back as it was read, got:\n%s", got)
		}
	}
}

func TestParseRefusesAKindOutsideTheSet(t *testing.T) {
	_, err := Parse("# Changelog\n\n## Unreleased\n\n### Improved\n\n- Faster.\n")
	if !errors.Is(err, ErrUnknownKind) {
		t.Errorf("expected ErrUnknownKind for ### Improved, got %v", err)
	}
}

func TestParseRefusesALogWithoutUnreleased(t *testing.T) {
	_, err := Parse("# Changelog\n\n## v0.1.0 (2026-01-01)\n\n- First.\n")
	if !errors.Is(err, ErrNoUnreleased) {
		t.Errorf("expected ErrNoUnreleased, got %v", err)
	}
}

func TestEntriesAnswersTheVersionsBody(t *testing.T) {
	got, err := mustParse(t, cut).Entries("v0.12.1")
	if err != nil {
		t.Fatalf("reading v0.12.1: %v", err)
	}
	want := "### Fixed\n\n- SRT holds its watch window once."
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
	if _, err := mustParse(t, cut).Entries("v0.9.0"); !errors.Is(err, ErrNoSuchVersion) {
		t.Errorf("expected ErrNoSuchVersion for a version the log lacks, got %v", err)
	}
}

func TestVersionsListsEveryCutVersionNewestFirst(t *testing.T) {
	got := strings.Join(mustParse(t, cut).Versions(), " ")
	if got != "v0.13.0 v0.12.1" {
		t.Errorf("expected v0.13.0 v0.12.1, got %s", got)
	}
}

func TestPreviousPicksTheNearestOlderTag(t *testing.T) {
	tags := []string{"v0.9.5", "v0.10.0", "v0.10.3", "v0.11.0", "nightly", "v0.12.0"}
	cases := map[string]string{"v0.11.0": "v0.10.3", "v0.10.0": "v0.9.5", "v0.13.0": "v0.12.0", "v0.9.5": ""}
	for version, want := range cases {
		if got := Previous(tags, version); got != want {
			t.Errorf("previous of %s: expected %q, got %q", version, want, got)
		}
	}
}

func TestNewestPicksTheHighestTag(t *testing.T) {
	if got := Newest([]string{"v0.9.5", "v0.10.0", "nightly", "v0.2.0"}); got != "v0.10.0" {
		t.Errorf("expected v0.10.0, got %q", got)
	}
	if got := Newest([]string{"nightly"}); got != "" {
		t.Errorf("expected no tag from tags carrying no version, got %q", got)
	}
}
