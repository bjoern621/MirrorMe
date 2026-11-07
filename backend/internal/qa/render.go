package qa

import (
	"fmt"
	"slices"
	"strings"

	"bjoernblessin.de/screenshare/internal/release"
)

// Matrix renders every record as qa/README.md, one section per release, newest first.
func Matrix(records []Record) string {
	var b strings.Builder
	b.WriteString("# Tested hardware\n\n")
	b.WriteString("Each release's test runs, one row per machine.\n")
	b.WriteString("`task qa:record` writes a run's record under `runs/` and redraws this page ([releasing](../docs/releasing.md)).\n")
	b.WriteString("A test count links its record, which names every failure and every skip with its reason.\n")

	for _, version := range versions(records) {
		var rows []Record
		for _, r := range records {
			if r.Version == version {
				rows = append(rows, r)
			}
		}
		b.WriteString("\n## " + version + "\n\n" + Table(rows, ""))
	}
	return b.String()
}

// versions answers every version that the records name, newest first.
func versions(records []Record) []string {
	var out []string
	for _, r := range records {
		if !slices.Contains(out, r.Version) {
			out = append(out, r.Version)
		}
	}
	slices.SortFunc(out, func(a, b string) int {
		if a == b {
			return 0
		}
		if release.Compare(a, b) == release.StateBehind {
			return 1
		}
		return -1
	})
	return out
}

// Table renders records as a Markdown table, one row per machine, sorted by machine name.
// Each test count links its record at linkBase + Record.Path.
func Table(records []Record, linkBase string) string {
	rows := slices.Clone(records)
	slices.SortFunc(rows, func(a, b Record) int { return strings.Compare(a.Machine.Name, b.Machine.Name) })

	var b strings.Builder
	b.WriteString("| Machine | OS | CPU | GPU | Media stack | Tests | Checklist |\n")
	b.WriteString("| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, r := range rows {
		m := r.Machine
		cells := []string{
			m.Name,
			join(", ", m.OS, prefixed("kernel ", m.Kernel), m.Arch),
			orDash(m.CPU),
			gpus(m.GPUs),
			orDash(join(", ", prefixed("GStreamer ", m.GStreamer), prefixed("FFmpeg ", m.FFmpeg), vaShort(m.VADriver))),
			fmt.Sprintf("[%d passed, %d failed, %d skipped](%s%s)", r.Tests.Passed, r.Tests.Failed, r.Tests.Skipped, linkBase, r.Path()),
			checklist(r.Checklist),
		}
		b.WriteString("| " + strings.Join(cells, " | ") + " |\n")
	}
	return b.String()
}

func gpus(list []GPU) string {
	var out []string
	for _, g := range list {
		detail := join(" ", g.Driver, g.DriverVersion)
		if detail != "" {
			out = append(out, g.Name+" ("+detail+")")
		} else {
			out = append(out, g.Name)
		}
	}
	return orDash(strings.Join(out, "<br>"))
}

func checklist(checks []Check) string {
	if checks == nil {
		return "not run"
	}
	counts := map[Result]int{}
	for _, c := range checks {
		counts[c.Result]++
	}
	return fmt.Sprintf("%d passed, %d failed, %d skipped", counts[Pass], counts[Fail], counts[Skipped])
}

// vaShort cuts the adapter off a VA vendor string, the GPU column naming it already:
// "Mesa Gallium driver 26.1.6 for AMD Radeon 780M Graphics (radeonsi)" reads "Mesa Gallium driver 26.1.6".
func vaShort(vendor string) string {
	short, _, _ := strings.Cut(vendor, " for ")
	return strings.TrimSpace(short)
}

func prefixed(prefix, value string) string {
	if value == "" {
		return ""
	}
	return prefix + value
}

func join(sep string, parts ...string) string {
	return strings.Join(slices.DeleteFunc(parts, func(p string) bool { return p == "" }), sep)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
