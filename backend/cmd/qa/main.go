// Command qa records a test run of a release on this machine and redraws qa/README.md from the records
// (docs/releasing.md).
//
//	qa record [-version v] [-machine name]
//	qa matrix [-check]
//
// record runs the release's own hardware suite from its tag, reads this machine's hardware
// and commits the record with the page.
// matrix redraws the page, or with -check fails where the page differs from the records.
// Every failure here comes from the checkout or the machine (an Umgebungsfehler) and exits with status 1.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"bjoernblessin.de/mirrorme/internal/qa"
)

func main() {
	if len(os.Args) < 2 {
		fail(fmt.Errorf("name a step: record or matrix"))
	}
	root, err := git("", "rev-parse", "--show-toplevel")
	if err != nil {
		fail(err)
	}

	flags := flag.NewFlagSet(os.Args[1], flag.ExitOnError)
	switch os.Args[1] {
	case "record":
		version := flags.String("version", "", "release tag to test (default: the newest tag)")
		machine := flags.String("machine", os.Getenv("QA_MACHINE"), "name that this machine's records carry (default: QA_MACHINE, else the host name)")
		_ = flags.Parse(os.Args[2:])
		err = record(root, *version, *machine)
	case "matrix":
		check := flags.Bool("check", false, "fail where qa/README.md differs from the records, writing nothing")
		_ = flags.Parse(os.Args[2:])
		err = matrix(root, *check)
	default:
		err = fmt.Errorf("unknown step %q: record or matrix", os.Args[1])
	}
	if err != nil {
		fail(err)
	}
}

// matrix redraws qa/README.md from every record under qa/runs.
func matrix(root string, check bool) error {
	dir := filepath.Join(root, "qa")
	records, err := qa.Load(dir)
	if err != nil {
		return err
	}
	page := []byte(qa.Matrix(records))
	path := filepath.Join(dir, "README.md")

	if !check {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		return os.WriteFile(path, page, 0o644)
	}
	current, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(current, page) {
		return fmt.Errorf("qa/README.md differs from the records under qa/runs: run `task qa:matrix`")
	}
	return nil
}

func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "qa:", err)
	os.Exit(1)
}
