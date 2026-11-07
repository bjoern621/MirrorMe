package main

import "testing"

func TestNvidiaReleaseReadsBothKernelModules(t *testing.T) {
	for _, c := range []struct{ module, version, want string }{
		{"closed", "NVRM version: NVIDIA UNIX x86_64 Kernel Module  550.54.14  Thu Feb 22 01:44:30 UTC 2024\nGCC version:  gcc version 13.2.0 (GCC) \n", "550.54.14"},
		{"open", "NVRM version: NVIDIA UNIX Open Kernel Module for x86_64  615.78.08  Release Build  (nixbld@)  \nGCC version:  gcc version 16.2.0 (GCC) \n", "615.78.08"},
	} {
		if got := nvidiaRelease(c.version); got != c.want {
			t.Errorf("%s module: expected %q, got %q", c.module, c.want, got)
		}
	}
}
