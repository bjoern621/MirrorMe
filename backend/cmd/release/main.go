// Command release renders the GitHub release text that the notes workflow writes (docs/releasing.md).
//
//	release notes <v>         the GitHub release text for v, with the machines that tested it
//	release versions          every version CHANGELOG.md holds, newest first
//
// Each prints its answer to stdout and nothing else, so a script can take it whole.
// Every failure here comes from the file or the checkout (an Umgebungsfehler) and exits with status 1.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"bjoernblessin.de/mirrorme/internal/changelog"
	"bjoernblessin.de/mirrorme/internal/qa"
	"bjoernblessin.de/mirrorme/internal/release"
)

const repoURL = "https://github.com/" + release.Repo

func main() {
	root := flag.String("root", "", "repository root (default: the checkout that this runs in)")
	flag.Parse()

	if err := run(*root, flag.Args()); err != nil {
		fmt.Fprintln(os.Stderr, "release:", err)
		os.Exit(1)
	}
}

func run(root string, args []string) error {
	if root == "" {
		top, err := git("", "rev-parse", "--show-toplevel")
		if err != nil {
			return err
		}
		root = top
	}
	if len(args) == 0 {
		return fmt.Errorf("name a step: notes or versions")
	}

	log, err := readLog(root)
	if err != nil {
		return err
	}
	switch step := args[0]; {
	case step == "notes" && len(args) == 2:
		notes, err := notes(root, log, args[1])
		if err != nil {
			return err
		}
		fmt.Print(notes)
	case step == "versions" && len(args) == 1:
		for _, v := range log.Versions() {
			fmt.Println(v)
		}
	default:
		return fmt.Errorf("unknown step %q: notes <v> or versions", strings.Join(args, " "))
	}
	return nil
}

// notes renders a release's GitHub text: its changelog entries, the machines that tested it
// and the commits since the release before it.
func notes(root string, log changelog.Log, version string) (string, error) {
	entries, err := log.Entries(version)
	if err != nil {
		return "", err
	}
	records, err := qa.Load(filepath.Join(root, "qa"))
	if err != nil {
		return "", err
	}
	tags, err := tags(root)
	if err != nil {
		return "", err
	}

	var tested []qa.Record
	for _, r := range records {
		if r.Version == version {
			tested = append(tested, r)
		}
	}

	var b strings.Builder
	b.WriteString(entries + "\n\n## Tested on\n\n")
	if len(tested) == 0 {
		b.WriteString("No machine has recorded a run of this release ([qa](" + repoURL + "/tree/main/qa)).\n")
	} else {
		b.WriteString(qa.Table(tested, repoURL+"/blob/main/qa/"))
	}
	if previous := changelog.Previous(tags, version); previous != "" {
		b.WriteString("\n**Full Changelog**: " + repoURL + "/compare/" + previous + "..." + version + "\n")
	}
	return b.String(), nil
}

func readLog(root string) (changelog.Log, error) {
	data, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		return changelog.Log{}, err
	}
	log, err := changelog.Parse(string(data))
	if err != nil {
		return changelog.Log{}, fmt.Errorf("CHANGELOG.md: %w", err)
	}
	return log, nil
}

func tags(root string) ([]string, error) {
	out, err := git(root, "tag", "--list", "v*")
	if err != nil {
		return nil, err
	}
	return strings.Fields(out), nil
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
