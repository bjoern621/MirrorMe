package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"bjoernblessin.de/screenshare/internal/qa"
)

// item names a check that needs a person at the running app.
// The ID must stay fixed once records carry it. The text may change.
type item struct {
	id   string
	text string
}

// checklist lists the items that qa record asks after the suite, in this order.
var checklist = []item{
	{"publish", "Publish a screen. A second machine watching sees it move within a few seconds."},
	{"encoder", "Publish once through each hardware encoder that Setup offers on this machine. Each one reaches the viewer."},
	{"watch", "Watch two streams at once. Both tiles play without stalls."},
	{"audio", "Publish with desktop audio. The viewer hears it."},
	{"restart", "Stop the publish and start it again. The viewer's tile plays again by itself."},
	{"update", "Install the release before this one and let the app update. It restarts on this release."},
}

// ask walks the checklist against version on in and out.
// A failed item takes a note on what failed.
// Errors if in ends before every item has an answer.
func ask(version string, in io.Reader, out io.Writer) ([]qa.Check, error) {
	lines := bufio.NewScanner(in)
	fmt.Fprintf(out, "\nCheck each item in the app running %s.\n", version)

	var checks []qa.Check
	for i, it := range checklist {
		fmt.Fprintf(out, "\n%d/%d %s\n", i+1, len(checklist), it.text)
		result, err := answer(lines, out)
		if err != nil {
			return nil, err
		}
		check := qa.Check{ID: it.id, Result: result}
		if result == qa.Fail {
			fmt.Fprint(out, "What failed? ")
			if !lines.Scan() {
				return nil, errors.New("the checklist ended before its note")
			}
			check.Note = strings.TrimSpace(lines.Text())
		}
		checks = append(checks, check)
	}
	return checks, nil
}

func answer(lines *bufio.Scanner, out io.Writer) (qa.Result, error) {
	for {
		fmt.Fprint(out, "[p]ass, [f]ail or [s]kip: ")
		if !lines.Scan() {
			return "", errors.New("the checklist ended before every item had an answer")
		}
		switch strings.ToLower(strings.TrimSpace(lines.Text())) {
		case "p", "pass":
			return qa.Pass, nil
		case "f", "fail":
			return qa.Fail, nil
		case "s", "skip":
			return qa.Skipped, nil
		}
	}
}
