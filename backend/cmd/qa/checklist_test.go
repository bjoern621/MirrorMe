package main

import (
	"io"
	"strings"
	"testing"

	"bjoernblessin.de/screenshare/internal/qa"
)

func TestAskTakesANoteOnAFailure(t *testing.T) {
	in := strings.NewReader("p\nmaybe\nf\nno sound\ns\npass\nP\np\n")
	checks, err := ask("v0.12.1", in, io.Discard)
	if err != nil {
		t.Fatalf("asking: %v", err)
	}
	if len(checks) != len(checklist) {
		t.Fatalf("expected an answer per item, got %d of %d", len(checks), len(checklist))
	}
	if checks[1] != (qa.Check{ID: checklist[1].id, Result: qa.Fail, Note: "no sound"}) {
		t.Errorf("expected the second item failed with its note, got %+v", checks[1])
	}
	if checks[2].Result != qa.Skipped || checks[4].Result != qa.Pass {
		t.Errorf("expected skip on the third and pass on the fifth, got %+v", checks)
	}
}

func TestAskRefusesAnInputEndingEarly(t *testing.T) {
	if _, err := ask("v0.12.1", strings.NewReader("p\n"), io.Discard); err == nil {
		t.Errorf("an input ending before the last item reports an error")
	}
}
