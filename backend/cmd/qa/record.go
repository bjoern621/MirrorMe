package main

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"bjoernblessin.de/mirrorme/internal/changelog"
	"bjoernblessin.de/mirrorme/internal/qa"
)

// record tests version on this machine and commits the record beside the redrawn page.
// The suite runs from the tag's own tree, exported to a temporary directory,
// so the record answers for the release whatever the checkout holds.
func record(root, version, machine string) error {
	if version == "" {
		out, err := git(root, "tag", "--list", "v*")
		if err != nil {
			return err
		}
		version = changelog.Newest(strings.Fields(out))
		if version == "" {
			return errors.New("the checkout holds no vX.Y.Z tag: fetch the tags or pass -version")
		}
	}
	if _, err := git(root, "rev-parse", "--verify", "--quiet", "refs/tags/"+version); err != nil {
		return fmt.Errorf("no tag %s in this checkout (git fetch --tags?)", version)
	}
	if machine == "" {
		host, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("no machine name given and no host name to take: %w", err)
		}
		machine = host
	}

	tree, err := os.MkdirTemp("", "mirrorme-qa-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tree)

	fmt.Fprintf(os.Stderr, "Exporting %s to %s\n", version, tree)
	if err := export(root, version, tree); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Running the hardware suite of %s\n", version)
	tests, err := runSuite(filepath.Join(tree, "backend"))
	if err != nil {
		return err
	}

	r := qa.Record{Version: version, Date: time.Now().Format(time.DateOnly), Machine: detect(machine), Tests: tests}
	if err := r.Write(filepath.Join(root, "qa")); err != nil {
		return err
	}
	if err := matrix(root, false); err != nil {
		return err
	}
	return commit(root, r)
}

// export writes the tree of tag into dir.
func export(root, tag, dir string) error {
	cmd := exec.Command("git", "archive", "--format=tar", tag)
	cmd.Dir = root
	cmd.Stderr = os.Stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("git archive %s: %w", tag, err)
	}
	if err := untar(out, dir); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("exporting %s: %w", tag, err)
	}
	// The tar reader stops at the end marker, and git blocks on a full pipe until its padding gets read.
	if _, err := io.Copy(io.Discard, out); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return fmt.Errorf("exporting %s: %w", tag, err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("git archive %s: %w", tag, err)
	}
	return nil
}

func untar(r io.Reader, dir string) error {
	archive := tar.NewReader(r)
	for {
		h, err := archive.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		path := filepath.Join(dir, filepath.FromSlash(h.Name))
		switch h.Typeflag {
		case tar.TypeDir:
			err = os.MkdirAll(path, 0o755)
		case tar.TypeReg:
			err = writeFile(path, archive, os.FileMode(h.Mode).Perm())
		case tar.TypeSymlink:
			err = os.Symlink(h.Linkname, path)
		}
		if err != nil {
			return err
		}
	}
}

func writeFile(path string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, r); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// runSuite runs every backend test with the hardware tag and counts the answers.
// A failing test goes into the record.
// Errors if go test fails to start, exits abnormally or prints no event.
func runSuite(backend string) (qa.Tests, error) {
	cmd := exec.Command("go", "test", "-json", "-tags", "hardware", "./...")
	cmd.Dir = backend
	cmd.Env = append(os.Environ(), "CGO_ENABLED=1")
	cmd.Stderr = os.Stderr
	out, err := cmd.StdoutPipe()
	if err != nil {
		return qa.Tests{}, err
	}
	if err := cmd.Start(); err != nil {
		return qa.Tests{}, fmt.Errorf("go test: %w", err)
	}

	tests, err := qa.Summarize(out, progress)
	waitErr := cmd.Wait()
	if err != nil {
		return qa.Tests{}, fmt.Errorf("go test: %w (%v)", err, waitErr)
	}
	var exit *exec.ExitError
	if waitErr != nil && !errors.As(waitErr, &exit) {
		return qa.Tests{}, fmt.Errorf("go test: %w", waitErr)
	}
	return tests, nil
}

// progress prints one line per package as its tests finish.
func progress(e qa.Event) {
	if e.Test != "" || (e.Action != "pass" && e.Action != "fail" && e.Action != "skip") {
		return
	}
	fmt.Fprintf(os.Stderr, "  %-4s %s (%.0f s)\n", e.Action, e.Package, e.Elapsed)
}

// commit commits the record and the page, and nothing else staged beside them.
// A run whose record matches the committed one leaves no commit.
func commit(root string, r qa.Record) error {
	paths := []string{"qa/" + r.Path(), "qa/README.md"}
	if _, err := git(root, append([]string{"add", "--"}, paths...)...); err != nil {
		return err
	}
	if _, err := git(root, append([]string{"diff", "--cached", "--quiet", "--"}, paths...)...); err == nil {
		fmt.Fprintf(os.Stderr, "The record of %s on %s matches the committed one.\n", r.Version, r.Machine.Name)
		return nil
	}
	message := "Record a test run of " + r.Version + " on " + r.Machine.Name
	if _, err := git(root, append([]string{"commit", "--quiet", "-m", message, "--"}, paths...)...); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Committed qa/%s.\nPush main to put it on the release page.\n", r.Path())
	return nil
}
