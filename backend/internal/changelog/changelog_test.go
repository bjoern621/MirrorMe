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

func TestNextBumpsByTheKindsUnreleased(t *testing.T) {
	cases := []struct {
		name   string
		kinds  string
		latest string
		want   string
	}{
		{"a fix alone takes a patch", "### Fixed\n\n- One.\n", "v0.12.1", "v0.12.2"},
		{"an addition takes a minor", "### Added\n\n- One.\n", "v0.12.1", "v0.13.0"},
		{"a change takes a minor", "### Changed\n\n- One.\n", "v0.12.1", "v0.13.0"},
		{"a removal before 1.0 takes a minor", "### Removed\n\n- One.\n", "v0.12.1", "v0.13.0"},
		{"a removal from 1.0 takes a major", "### Removed\n\n- One.\n", "v1.4.2", "v2.0.0"},
		{"a fix beside an addition takes a minor", "### Fixed\n\n- One.\n\n### Added\n\n- Two.\n", "v0.12.1", "v0.13.0"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			log := mustParse(t, "# Changelog\n\n## Unreleased\n\n"+c.kinds)
			got, err := log.Next(c.latest)
			if err != nil {
				t.Fatalf("next after %s: %v", c.latest, err)
			}
			if got != c.want {
				t.Errorf("next after %s: expected %s, got %s", c.latest, c.want, got)
			}
		})
	}
}

func TestNextRefusesAnEmptyUnreleased(t *testing.T) {
	_, err := mustParse(t, cut).Next("v0.13.0")
	if !errors.Is(err, ErrNothingUnreleased) {
		t.Errorf("expected ErrNothingUnreleased, got %v", err)
	}
}

func TestCutMovesUnreleasedUnderTheVersion(t *testing.T) {
	got, err := mustParse(t, pending).Cut("v0.13.0", "2026-11-07")
	if err != nil {
		t.Fatalf("cutting: %v", err)
	}
	if got.String() != cut {
		t.Errorf("expected:\n%s\ngot:\n%s", cut, got.String())
	}
}

func TestCutTwiceEqualsCutOnce(t *testing.T) {
	got, err := mustParse(t, cut).Cut("v0.13.0", "2026-11-08")
	if err != nil {
		t.Fatalf("cutting a version already cut: %v", err)
	}
	if got.String() != cut {
		t.Errorf("a second cut changes nothing, got:\n%s", got.String())
	}
}

func TestCutRefuses(t *testing.T) {
	cutThenMore := strings.Replace(cut, "## Unreleased\n", "## Unreleased\n\n### Fixed\n\n- Later.\n", 1)
	cases := []struct {
		name    string
		text    string
		version string
		want    error
	}{
		{"an empty Unreleased", cut, "v0.14.0", ErrNothingUnreleased},
		{"a version already cut with entries pending", cutThenMore, "v0.13.0", ErrAlreadyCut},
		{"a version not past the newest", pending, "v0.12.0", ErrNotNewer},
		{"a string with no version", pending, "next", ErrNotAVersion},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := mustParse(t, c.text).Cut(c.version, "2026-11-07")
			if !errors.Is(err, c.want) {
				t.Errorf("expected %v, got %v", c.want, err)
			}
		})
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
