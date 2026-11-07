package timeline

import (
	"testing"
	"time"

	mirrormev1 "bjoernblessin.de/mirrorme/api/gen/go/mirrorme/v1"
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
	tl.Sample(59.8)
	tl.Sample(60)

	if got := tl.Snapshot(); got.Level != mirrormev1.StreamVerdict_LEVEL_CLEAN {
		t.Fatalf("a run at its target is clean, got %v", got.Level)
	}
}

func TestOneSlowSampleIsNoDip(t *testing.T) {
	tl := started()
	tl.Sample(30)
	tl.Sample(60)

	if got := tl.Snapshot(); got.Level != mirrormev1.StreamVerdict_LEVEL_CLEAN {
		t.Fatalf("a single slow reading is noise, got %v", got.Level)
	}
}

func TestASustainedDropIsADipThatDisappoints(t *testing.T) {
	tl := started()
	tl.Sample(40)
	tl.Sample(30)

	got := tl.Snapshot()
	if got.Level != mirrormev1.StreamVerdict_LEVEL_DISAPPOINTS {
		t.Fatalf("a dip disappoints, got %v", got.Level)
	}
	if got.Cause.GetCode() != mirrormev1.TextCode_TEXT_CODE_VERDICT_DIP {
		t.Fatalf("the cause names the dip, got %v", got.Cause.GetCode())
	}
	if args := got.Cause.GetArgs(); len(args) != 2 || args[0].GetDecimal() != 30 || args[1].GetDecimal() != 60 {
		t.Fatalf("the dip carries its lowest rate and its target, got %+v", args)
	}
}

func TestADipEndsOnceTheRateRecovers(t *testing.T) {
	tl := started()
	tl.Sample(40)
	tl.Sample(40)
	tl.Sample(58)

	if got := tl.Snapshot(); got.Level != mirrormev1.StreamVerdict_LEVEL_CLEAN {
		t.Fatalf("a recovered run is clean, got %v", got.Level)
	}
}

func TestARestartDisappointsUntilFramesFlowAgain(t *testing.T) {
	tl := started()
	cause := &mirrormev1.Text{Code: mirrormev1.TextCode_TEXT_CODE_TRANSPORT_FALLING_BACK}
	tl.Restart(cause)

	got := tl.Snapshot()
	if got.Level != mirrormev1.StreamVerdict_LEVEL_DISAPPOINTS || got.Cause != cause {
		t.Fatalf("a restarting run disappoints for the exit's cause, got %v %v", got.Level, got.Cause)
	}

	tl.Sample(60)
	if got := tl.Snapshot(); got.Level != mirrormev1.StreamVerdict_LEVEL_CLEAN {
		t.Fatalf("frames flowing again end the restart, got %v", got.Level)
	}
}

func TestAStopCarriesTheExitAndOutranksEverything(t *testing.T) {
	tl := started()
	tl.Sample(10)
	tl.Sample(10)
	tl.Stop(nil, "encoder gave up", "/var/log/run.log")

	got := tl.Snapshot()
	if got.Level != mirrormev1.StreamVerdict_LEVEL_STOPPED || got.Message != "encoder gave up" || got.LogPath != "/var/log/run.log" {
		t.Fatalf("a stop names what ended the run, got %+v", got)
	}
}

func TestBeginningARunClearsTheLastOne(t *testing.T) {
	tl := started()
	tl.Sample(10)
	tl.Sample(10)
	tl.Stop(nil, "x", "")
	tl.Begin(at(10), 30)

	got := tl.Snapshot()
	if got.Level != mirrormev1.StreamVerdict_LEVEL_CLEAN || got.StartedAt != at(10) {
		t.Fatalf("a new run starts clean, got %+v", got)
	}
}

func TestSamplesBeforeAnyRunChangeNothing(t *testing.T) {
	tl := New()
	tl.Sample(1)
	tl.Sample(1)

	if got := tl.Snapshot(); got.Level != mirrormev1.StreamVerdict_LEVEL_CLEAN {
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
		if got := tl.Sample(step.fps); got != step.moved {
			t.Fatalf("sample %d at %v fps: moved %v, want %v", i, step.fps, got, step.moved)
		}
	}
}
