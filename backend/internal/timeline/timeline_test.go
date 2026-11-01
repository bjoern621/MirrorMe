package timeline

import (
	"testing"
	"time"

	screensharev1 "bjoernblessin.de/screenshare/api/gen/go/screenshare/v1"
)

var t0 = time.Unix(1_800_000_000, 0)

func at(seconds int) time.Time { return t0.Add(time.Duration(seconds) * time.Second) }

func started() *Timeline {
	tl := New()
	tl.Begin(at(0), 60)
	return tl
}

func TestARunHoldingItsRateIsClean(t *testing.T) {
	tl := started()
	tl.Sample(at(1), 59.8)
	tl.Sample(at(2), 60)

	got := tl.Snapshot()
	if got.Level != screensharev1.StreamVerdict_LEVEL_CLEAN || len(got.Events) != 0 {
		t.Fatalf("a run at its target is clean with nothing to mark, got %v with %d events", got.Level, len(got.Events))
	}
}

func TestOneSlowSampleIsNoDip(t *testing.T) {
	tl := started()
	tl.Sample(at(1), 30)
	tl.Sample(at(2), 60)

	if got := tl.Snapshot(); len(got.Events) != 0 {
		t.Fatalf("a single slow reading is noise, got %d events", len(got.Events))
	}
}

func TestASustainedDropIsADipThatDisappoints(t *testing.T) {
	tl := started()
	tl.Sample(at(1), 40)
	tl.Sample(at(2), 30)

	got := tl.Snapshot()
	if got.Level != screensharev1.StreamVerdict_LEVEL_DISAPPOINTS {
		t.Fatalf("a dip disappoints, got %v", got.Level)
	}
	if got.Cause.GetCode() != screensharev1.TextCode_TEXT_CODE_VERDICT_DIP {
		t.Fatalf("the cause names the dip, got %v", got.Cause.GetCode())
	}
	if len(got.Events) != 1 || got.Events[0].Kind != screensharev1.StreamEvent_KIND_DIP {
		t.Fatalf("one dip event, got %+v", got.Events)
	}
	if got.Events[0].Fps != 30 || got.Events[0].TargetFps != 60 {
		t.Fatalf("the dip carries its lowest rate and its target, got %+v", got.Events[0])
	}
}

func TestADipEndsOnceTheRateRecovers(t *testing.T) {
	tl := started()
	tl.Sample(at(1), 40)
	tl.Sample(at(2), 40)
	tl.Sample(at(3), 58)

	got := tl.Snapshot()
	if got.Level != screensharev1.StreamVerdict_LEVEL_CLEAN || len(got.Events) != 1 {
		t.Fatalf("a recovered run is clean and keeps the dip it had, got %v with %d events", got.Level, len(got.Events))
	}
}

func TestARestartDisappointsUntilFramesFlowAgain(t *testing.T) {
	tl := started()
	cause := &screensharev1.Text{Code: screensharev1.TextCode_TEXT_CODE_TRANSPORT_FALLING_BACK}
	tl.Restart(at(5), cause)

	got := tl.Snapshot()
	if got.Level != screensharev1.StreamVerdict_LEVEL_DISAPPOINTS || got.Cause != cause {
		t.Fatalf("a restarting run disappoints for the exit's cause, got %v %v", got.Level, got.Cause)
	}

	tl.Sample(at(8), 60)
	if got := tl.Snapshot(); got.Level != screensharev1.StreamVerdict_LEVEL_CLEAN {
		t.Fatalf("frames flowing again end the restart, got %v", got.Level)
	}
}

func TestAStopCarriesTheExitAndOutranksEverything(t *testing.T) {
	tl := started()
	tl.Sample(at(1), 10)
	tl.Sample(at(2), 10)
	tl.Stop(at(3), nil, "encoder gave up", "/var/log/run.log")

	got := tl.Snapshot()
	if got.Level != screensharev1.StreamVerdict_LEVEL_STOPPED || got.Message != "encoder gave up" || got.LogPath != "/var/log/run.log" {
		t.Fatalf("a stop names what ended the run, got %+v", got)
	}
}

func TestMembersArrivingAndLeavingAreMarked(t *testing.T) {
	tl := started()
	tl.Members(at(1), []string{"ada"})
	tl.Members(at(2), []string{"ada", "lin"})
	tl.Members(at(3), []string{"lin"})

	got := tl.Snapshot().Events
	if len(got) != 2 {
		t.Fatalf("the first reading is the baseline, then one join and one leave, got %+v", got)
	}
	if got[0].Kind != screensharev1.StreamEvent_KIND_JOIN || got[0].Who != "lin" {
		t.Fatalf("lin joined, got %+v", got[0])
	}
	if got[1].Kind != screensharev1.StreamEvent_KIND_LEAVE || got[1].Who != "ada" {
		t.Fatalf("ada left, got %+v", got[1])
	}
}

func TestBeginningARunClearsTheLastOne(t *testing.T) {
	tl := started()
	tl.Sample(at(1), 10)
	tl.Sample(at(2), 10)
	tl.Stop(at(3), nil, "x", "")
	tl.Begin(at(10), 30)

	got := tl.Snapshot()
	if got.Level != screensharev1.StreamVerdict_LEVEL_CLEAN || len(got.Events) != 0 || got.StartedAt != at(10) {
		t.Fatalf("a new run starts clean, got %+v", got)
	}
}

func TestTheEventsKeepTheMostRecent(t *testing.T) {
	tl := started()
	for i := range MAX_EVENTS + 5 {
		tl.Restart(at(i), nil)
	}

	got := tl.Snapshot().Events
	if len(got) != MAX_EVENTS || !got[len(got)-1].At.Equal(at(MAX_EVENTS+4)) {
		t.Fatalf("the cap drops the oldest, got %d ending %v", len(got), got[len(got)-1].At)
	}
}

func TestSamplesBeforeAnyRunChangeNothing(t *testing.T) {
	tl := New()
	tl.Sample(at(1), 1)
	tl.Sample(at(2), 1)

	if got := tl.Snapshot(); got.Level != screensharev1.StreamVerdict_LEVEL_CLEAN || len(got.Events) != 0 {
		t.Fatalf("no run, nothing to judge, got %+v", got)
	}
}

func TestASampleReportsWhetherTheJudgmentMoved(t *testing.T) {
	tl := started()
	steps := []struct {
		fps   float64
		moved bool
	}{
		{60, false},
		{30, false},
		{30, true},
		{20, true},
		{25, false},
		{60, true},
		{60, false},
	}
	for i, step := range steps {
		if got := tl.Sample(at(i+1), step.fps); got != step.moved {
			t.Fatalf("sample %d at %v fps: moved %v, want %v", i, step.fps, got, step.moved)
		}
	}
}

func TestAMembersReadingReportsWhetherAnyoneCameOrWent(t *testing.T) {
	tl := started()
	if tl.Members(at(1), []string{"ada"}) {
		t.Fatalf("the baseline marks nobody")
	}
	if tl.Members(at(2), []string{"ada"}) {
		t.Fatalf("the same group marks nobody")
	}
	if !tl.Members(at(3), []string{"ada", "bo"}) {
		t.Fatalf("bo joining is a mark")
	}
}
