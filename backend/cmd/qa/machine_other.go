//go:build !linux && !windows

package main

import "bjoernblessin.de/mirrorme/internal/qa"

// platform reads nothing past the runtime on a system without a reader of its own here.
func platform() qa.Machine {
	return qa.Machine{}
}
