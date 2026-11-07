package qa

import (
	"strings"
	"testing"
)

const events = `{"Action":"run","Package":"bjoernblessin.de/mirrorme/internal/ffmpeg","Test":"TestGpu"}
{"Action":"output","Package":"bjoernblessin.de/mirrorme/internal/ffmpeg","Test":"TestGpu","Output":"=== RUN   TestGpu\n"}
{"Action":"output","Package":"bjoernblessin.de/mirrorme/internal/ffmpeg","Test":"TestGpu","Output":"    gpu_test.go:53: no ffmpeg ddagrab/nvenc row to cover\n"}
{"Action":"output","Package":"bjoernblessin.de/mirrorme/internal/ffmpeg","Test":"TestGpu","Output":"--- SKIP: TestGpu (0.00s)\n"}
{"Action":"skip","Package":"bjoernblessin.de/mirrorme/internal/ffmpeg","Test":"TestGpu"}
{"Action":"pass","Package":"bjoernblessin.de/mirrorme/internal/ffmpeg","Test":"TestArgs/vaapi"}
{"Action":"fail","Package":"bjoernblessin.de/mirrorme/internal/ffmpeg","Test":"TestArgs/nvenc"}
{"Action":"fail","Package":"bjoernblessin.de/mirrorme/internal/ffmpeg","Test":"TestArgs"}
{"Action":"pass","Package":"bjoernblessin.de/mirrorme/internal/ffmpeg","Test":"TestProbe"}
{"Action":"fail","Package":"bjoernblessin.de/mirrorme/internal/ffmpeg"}
{"Action":"fail","Package":"bjoernblessin.de/mirrorme/internal/receive"}
{"Action":"pass","Package":"bjoernblessin.de/mirrorme/internal/text"}
`

func TestSummarizeCountsLeafTests(t *testing.T) {
	got, err := Summarize(strings.NewReader(events), nil)
	if err != nil {
		t.Fatalf("summarizing: %v", err)
	}
	if got.Passed != 2 || got.Failed != 2 || got.Skipped != 1 {
		t.Errorf("expected 2 passed, 2 failed, 1 skipped, got %d, %d, %d", got.Passed, got.Failed, got.Skipped)
	}
	wantFailures := "ffmpeg.TestArgs/nvenc receive (failed outside a test)"
	if f := strings.Join(got.Failures, " "); f != wantFailures {
		t.Errorf("expected failures %q, got %q", wantFailures, f)
	}
	if len(got.Skips) != 1 || got.Skips[0] != (Skip{Test: "ffmpeg.TestGpu", Reason: "no ffmpeg ddagrab/nvenc row to cover"}) {
		t.Errorf("expected the skip with its reason, got %+v", got.Skips)
	}
}

func TestSummarizeRefusesAnEmptyRun(t *testing.T) {
	if _, err := Summarize(strings.NewReader(""), nil); err == nil {
		t.Errorf("a run with no events reports an error")
	}
}

func sample() []Record {
	amd := Machine{
		Name: "desk", OS: "NixOS 26.05", Kernel: "7.3.0", Arch: "amd64", CPU: "AMD Ryzen 7 8845HS",
		GPUs:      []GPU{{Name: "AMD Radeon 780M", Driver: "amdgpu"}},
		VADriver:  "Mesa Gallium driver 26.1.6 for AMD Radeon 780M Graphics (radeonsi, phoenix)",
		GStreamer: "1.28.7", FFmpeg: "9.0.2",
	}
	nv := Machine{
		Name: "tower", OS: "Windows 11 Pro 10.0.26100", Arch: "amd64", CPU: "Intel Core i7-12700K",
		GPUs: []GPU{{Name: "NVIDIA GeForce RTX 3070", DriverVersion: "32.0.15.6094"}},
	}
	return []Record{
		{Version: "v0.12.1", Date: "2026-11-06", Machine: nv, Tests: Tests{Passed: 300, Skipped: 20}},
		{Version: "v0.12.1", Date: "2026-11-06", Machine: amd, Tests: Tests{Passed: 310, Failed: 1, Skipped: 10}},
		{Version: "v0.13.0", Date: "2026-11-08", Machine: amd, Tests: Tests{Passed: 320}},
	}
}

func TestTableRowsOneRecordPerMachine(t *testing.T) {
	got := Table(sample()[:2], "")
	want := `| Machine | OS | CPU | GPU | Media stack | Tests |
| --- | --- | --- | --- | --- | --- |
| desk | NixOS 26.05, kernel 7.3.0, amd64 | AMD Ryzen 7 8845HS | AMD Radeon 780M (amdgpu) | GStreamer 1.28.7, FFmpeg 9.0.2, Mesa Gallium driver 26.1.6 | [310 passed, 1 failed, 10 skipped](runs/v0.12.1/desk.json) |
| tower | Windows 11 Pro 10.0.26100, amd64 | Intel Core i7-12700K | NVIDIA GeForce RTX 3070 (32.0.15.6094) | - | [300 passed, 0 failed, 20 skipped](runs/v0.12.1/tower.json) |
`
	if got != want {
		t.Errorf("expected:\n%s\ngot:\n%s", want, got)
	}
}

func TestMatrixGroupsByVersionNewestFirst(t *testing.T) {
	got := Matrix(sample())
	first := strings.Index(got, "## v0.13.0")
	second := strings.Index(got, "## v0.12.1")
	if first < 0 || second < 0 || first > second {
		t.Errorf("expected v0.13.0 above v0.12.1, got:\n%s", got)
	}
}

func TestPathNamesTheVersionAndMachine(t *testing.T) {
	r := Record{Version: "v0.12.1", Machine: Machine{Name: "Björn's Desk"}}
	if got := r.Path(); got != "runs/v0.12.1/bj-rn-s-desk.json" {
		t.Errorf("expected runs/v0.12.1/bj-rn-s-desk.json, got %s", got)
	}
}
