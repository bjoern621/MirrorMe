//go:build !windows

package app

import (
	"os/exec"
	goruntime "runtime"
)

func openWithDefaultHandler(path string) error {
	if goruntime.GOOS == "darwin" {
		return exec.Command("open", path).Run()
	}
	return exec.Command("xdg-open", path).Run()
}
