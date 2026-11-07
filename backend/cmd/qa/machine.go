package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"bjoernblessin.de/mirrorme/internal/qa"
)

// probeTimeout bounds each program asked about the machine, a driver init included.
const probeTimeout = 10 * time.Second

// detect reads this machine's hardware and media stack.
// A field that no probe answers stays empty.
// A warning names the failed read.
func detect(name string) qa.Machine {
	m := platform()
	m.Name = name
	m.Arch = runtime.GOARCH
	m.GStreamer = versionWord(output("gst-inspect-1.0", "--version"), "GStreamer ")
	m.FFmpeg = versionWord(output("ffmpeg", "-version"), "ffmpeg version ")
	return m
}

// versionWord answers the word after prefix on the first line of out holding it:
// "ffmpeg version 9.0.2 Copyright ..." reads "9.0.2".
func versionWord(out, prefix string) string {
	for line := range strings.SplitSeq(out, "\n") {
		if _, rest, found := strings.Cut(line, prefix); found {
			if fields := strings.Fields(rest); len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return ""
}

// output runs a program and answers its stdout, or "" with a warning where it fails.
func output(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, name, args...).Output()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: %s %s: %v, leaving it out of the record\n", name, strings.Join(args, " "), err)
		return ""
	}
	return string(out)
}
