package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"bjoernblessin.de/go-utils/util/assert"
)

// Replacing the directory an install lives in with the one an archive carries.
//
// Both release archives hold a single top-level directory named after the version,
// so an extract lands in a folder of its own
// (packaging/linux/package.sh, packaging/windows/package.ps1).
// That directory's contents are the install.
//
// The swap goes through renames beside the target rather than writing into it:
// a copy that stopped half way would leave an install that is neither release,
// and a rename either happened or did not.

// renameWait bounds the retries of a refused rename.
// A virus scanner or the search indexer holding one file open is enough for Windows to refuse
// the rename of the directory around it, and those let go within seconds.
const renameWait = 10 * time.Second

// outgoingWait bounds the applier's retries of an old tree that will not go.
const outgoingWait = 30 * time.Second

// retryStep is how often a refused rename or removal is tried again.
const retryStep = 250 * time.Millisecond

// outgoingSuffix names the old tree beside the install, from the swap until its removal.
const outgoingSuffix = ".outgoing"

func outgoingOf(target string) string {
	return target + outgoingSuffix
}

// retrying runs attempt until it succeeds or wait runs out, and answers its last error.
func retrying(wait time.Duration, attempt func() error) error {
	deadline := time.Now().Add(wait)
	for {
		err := attempt()
		if err == nil || time.Now().After(deadline) {
			return err
		}
		time.Sleep(retryStep)
	}
}

// renameRetrying renames from to to, trying again until renameWait runs out.
func renameRetrying(from, to string) error {
	return retrying(renameWait, func() error { return os.Rename(from, to) })
}

// removeRetrying removes the tree at path, trying again until wait runs out.
// A path that does not exist counts as removed.
func removeRetrying(path string, wait time.Duration) error {
	return retrying(wait, func() error { return os.RemoveAll(path) })
}

// Swap replaces target with what the archive carries.
//
// The staged tree is built beside target, so both renames are on one filesystem.
// A target whose parent this process cannot write is an Umgebungsfehler and leaves as an error,
// with the running install untouched.
//
// A file of the old tree that is still held leaves the tree beside target and the swap succeeds.
// The applier and later starts remove what stayed.
func Swap(archive, target string) error {
	assert.Assert(archive != "", "a swap names the archive it installs")
	assert.Assert(target != "", "a swap names the directory it replaces")

	parent := filepath.Dir(target)
	incoming, err := os.MkdirTemp(parent, ".mirrorme-incoming-")
	if err != nil {
		return fmt.Errorf("cannot stage beside %s: %w", target, err)
	}
	defer func() { _ = os.RemoveAll(incoming) }()

	if err := extract(archive, incoming); err != nil {
		return err
	}

	root, err := singleDir(incoming)
	if err != nil {
		return err
	}

	// A leftover goes first, since a rename onto a directory that holds files is refused.
	outgoing := outgoingOf(target)
	if err := removeRetrying(outgoing, renameWait); err != nil {
		return fmt.Errorf("cannot clear the old tree at %s: %w", outgoing, err)
	}

	// The old tree moves aside rather than being deleted:
	// a rename that fails leaves the install standing, and a delete would not.
	if err := renameRetrying(target, outgoing); err != nil {
		return fmt.Errorf("cannot move %s aside: %w", target, err)
	}

	if err := renameRetrying(root, target); err != nil {
		// Put back what was there, so a failure here is a run that changed nothing.
		_ = os.Rename(outgoing, target)
		return fmt.Errorf("cannot put %s in place: %w", target, err)
	}

	// One attempt, to keep a held file from delaying the relaunch.
	_ = os.RemoveAll(outgoing)
	return nil
}

// singleDir answers the one directory an extract produced.
func singleDir(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	var found string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if found != "" {
			return "", fmt.Errorf("the archive carries more than one top-level directory")
		}
		found = filepath.Join(dir, e.Name())
	}
	if found == "" {
		return "", fmt.Errorf("the archive carries no top-level directory")
	}
	return found, nil
}

// extract unpacks one release archive into dir, by what the file is called.
func extract(archive, dir string) error {
	switch {
	case strings.HasSuffix(archive, ".tar.gz"):
		return untar(archive, dir)
	case strings.HasSuffix(archive, ".zip"):
		return unzip(archive, dir)
	default:
		return fmt.Errorf("%s is not an archive this app unpacks", filepath.Base(archive))
	}
}

func untar(archive, dir string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	zipped, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer func() { _ = zipped.Close() }()

	reader := tar.NewReader(zipped)
	for {
		header, err := reader.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		path, err := within(dir, header.Name)
		if err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			// The mode travels: the two binaries in the archive are executable and the rest is not.
			if err := write(path, reader, header.FileInfo().Mode()); err != nil {
				return err
			}
		}
	}
}

func unzip(archive, dir string) error {
	reader, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()

	for _, entry := range reader.File {
		path, err := within(dir, entry.Name)
		if err != nil {
			return err
		}

		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			continue
		}

		body, err := entry.Open()
		if err != nil {
			return err
		}
		err = write(path, body, entry.Mode())
		_ = body.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// within resolves one archive entry under dir, and refuses a name that climbs out of it.
// An archive is somebody else's file, so a path in it is checked rather than joined.
func within(dir, name string) (string, error) {
	path := filepath.Join(dir, filepath.FromSlash(name))

	if !strings.HasPrefix(path, filepath.Clean(dir)+string(os.PathSeparator)) {
		return "", fmt.Errorf("the archive names a path outside itself: %s", name)
	}
	return path, nil
}

func write(path string, body io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}

	_, err = io.Copy(file, body)
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	return err
}
