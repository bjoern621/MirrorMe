package watchsvc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"bjoernblessin.de/mirrorme/internal/token"
)

// fixture is a service whose clock a test moves, with a signer standing in for groupd.
type fixture struct {
	t       *testing.T
	signer  *token.Signer
	service *Service
	now     time.Time
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	signer, err := token.NewSigner()
	if err != nil {
		t.Fatalf("drawing a signing key: %v", err)
	}
	keys, err := token.ParseKeySet(signer.JWKS())
	if err != nil {
		t.Fatalf("reading the signer's key set: %v", err)
	}
	f := &fixture{t: t, signer: signer, now: time.Unix(1_800_000_000, 0)}
	f.service = New(keys)
	f.service.now = func() time.Time { return f.now }
	return f
}

// tokenFor signs the token groupd hands a member of group G who publishes under publishPrefix.
func (f *fixture) tokenFor(subject, publishPrefix string) string {
	f.t.Helper()
	signed, err := f.signer.Sign(subject, token.GroupPermissions("G/", publishPrefix), f.now, 5*time.Minute)
	if err != nil {
		f.t.Fatalf("signing: %v", err)
	}
	return signed
}

func (f *fixture) call(method, target, bearer, body string) (int, string) {
	f.t.Helper()
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if bearer != "" {
		request.Header.Set("Authorization", "Bearer "+bearer)
	}
	recorder := httptest.NewRecorder()
	f.service.Handler("test").ServeHTTP(recorder, request)
	return recorder.Code, recorder.Body.String()
}

func (f *fixture) state(bearer, body string) int {
	f.t.Helper()
	status, _ := f.call(http.MethodPut, "/viewers", bearer, body)
	return status
}

func (f *fixture) viewersOf(bearer, path string) (int, []Viewer) {
	f.t.Helper()
	status, body := f.call(http.MethodGet, "/viewers?path="+url.QueryEscape(path), bearer, "")
	if status != http.StatusOK {
		return status, nil
	}
	var answer struct {
		Viewers []Viewer `json:"viewers"`
	}
	if err := json.Unmarshal([]byte(body), &answer); err != nil {
		f.t.Fatalf("reading %s: %v", body, err)
	}
	return status, answer.Viewers
}

const watchingBobs = `{"viewing":[{"path":"G/Bob/screen","transport":"rtsp","pathMs":41.5,"shownFps":60}]}`

func TestAStatementReachesThePublisherOfThatPath(t *testing.T) {
	f := newFixture(t)
	alice, bob := f.tokenFor("alice", "G/Alice/"), f.tokenFor("bob", "G/Bob/")

	if status := f.state(alice, watchingBobs); status != http.StatusNoContent {
		t.Fatalf("stating a viewing answered %d, want 204", status)
	}

	status, viewers := f.viewersOf(bob, "G/Bob/screen")
	if status != http.StatusOK {
		t.Fatalf("the publisher reading its viewers answered %d, want 200", status)
	}
	if len(viewers) != 1 || viewers[0].Member != "alice" || viewers[0].Transport != "rtsp" {
		t.Fatalf("viewers %+v, want alice over rtsp", viewers)
	}
	if viewers[0].PathMs == nil || *viewers[0].PathMs != 41.5 {
		t.Errorf("pathMs %v, want 41.5", viewers[0].PathMs)
	}
	if viewers[0].TotalMs != nil {
		t.Errorf("totalMs %v, want absent as stated", *viewers[0].TotalMs)
	}
}

func TestOnlyThePublisherReadsAStreamsViewers(t *testing.T) {
	f := newFixture(t)
	alice, carol := f.tokenFor("alice", "G/Alice/"), f.tokenFor("carol", "G/Carol/")
	f.state(alice, watchingBobs)

	if status, _ := f.viewersOf(carol, "G/Bob/screen"); status != http.StatusForbidden {
		t.Errorf("another member reading Bob's viewers answered %d, want 403", status)
	}
}

func TestAStatementOutsideTheTokensGroupStoresNothing(t *testing.T) {
	f := newFixture(t)
	alice, bob := f.tokenFor("alice", "G/Alice/"), f.tokenFor("bob", "G/Bob/")

	mixed := `{"viewing":[
		{"path":"G/Bob/screen","transport":"rtsp"},
		{"path":"H/Bob/screen","transport":"rtsp"}]}`
	if status := f.state(alice, mixed); status != http.StatusForbidden {
		t.Fatalf("a statement naming another group's path answered %d, want 403", status)
	}
	if _, viewers := f.viewersOf(bob, "G/Bob/screen"); len(viewers) != 0 {
		t.Errorf("a refused statement left %+v behind", viewers)
	}
}

func TestARestatementReplacesTheWholeSet(t *testing.T) {
	f := newFixture(t)
	alice, bob := f.tokenFor("alice", "G/Alice/"), f.tokenFor("bob", "G/Bob/")
	f.state(alice, watchingBobs)

	if status := f.state(alice, `{"viewing":[]}`); status != http.StatusNoContent {
		t.Fatalf("stating an empty set answered %d, want 204", status)
	}
	if _, viewers := f.viewersOf(bob, "G/Bob/screen"); len(viewers) != 0 {
		t.Errorf("after watching nothing, Bob still reads %+v", viewers)
	}
}

func TestAStatementTwiceIsOneViewer(t *testing.T) {
	f := newFixture(t)
	alice, bob := f.tokenFor("alice", "G/Alice/"), f.tokenFor("bob", "G/Bob/")
	f.state(alice, watchingBobs)
	f.state(alice, watchingBobs)

	if _, viewers := f.viewersOf(bob, "G/Bob/screen"); len(viewers) != 1 {
		t.Errorf("one member stating twice reads as %d viewers, want 1", len(viewers))
	}
}

func TestAStatementLapsesAfterTheLease(t *testing.T) {
	f := newFixture(t)
	alice := f.tokenFor("alice", "G/Alice/")
	f.state(alice, watchingBobs)

	f.now = f.now.Add(Lease + time.Second)
	bob := f.tokenFor("bob", "G/Bob/")
	if _, viewers := f.viewersOf(bob, "G/Bob/screen"); len(viewers) != 0 {
		t.Errorf("a statement nobody restated for a lease still reads %+v", viewers)
	}
}

func TestViewersArriveInMemberOrder(t *testing.T) {
	f := newFixture(t)
	for _, member := range []string{"dave", "alice", "carol"} {
		f.state(f.tokenFor(member, "G/"+member+"/"), watchingBobs)
	}

	_, viewers := f.viewersOf(f.tokenFor("bob", "G/Bob/"), "G/Bob/screen")
	var order []string
	for _, v := range viewers {
		order = append(order, v.Member)
	}
	if strings.Join(order, ",") != "alice,carol,dave" {
		t.Errorf("viewers arrive as %v, want alice,carol,dave", order)
	}
}

func TestACallWithoutAWorkingTokenIsUnauthorized(t *testing.T) {
	f := newFixture(t)
	expired := f.tokenFor("alice", "G/Alice/")
	f.now = f.now.Add(time.Hour)

	for _, bearer := range []string{"", "garbage", expired} {
		if status := f.state(bearer, watchingBobs); status != http.StatusUnauthorized {
			t.Errorf("stating with %q answered %d, want 401", bearer, status)
		}
		if status, _ := f.viewersOf(bearer, "G/Bob/screen"); status != http.StatusUnauthorized {
			t.Errorf("reading with %q answered %d, want 401", bearer, status)
		}
	}
}

func TestAMalformedStatementIsRefused(t *testing.T) {
	f := newFixture(t)
	alice := f.tokenFor("alice", "G/Alice/")

	for _, body := range []string{
		``,
		`{"viewing":[{"path":"","transport":"rtsp"}]}`,
		`{"viewing":[{"path":"G/Bob/screen","transport":""}]}`,
		`{"viewing":[{"path":"G/Bob/screen","transport":"rtsp","pathMs":-1}]}`,
		`{"viewing":[{"path":"G/Bob/screen","transport":"rtsp","droppedFps":-0.5}]}`,
	} {
		if status := f.state(alice, body); status != http.StatusBadRequest {
			t.Errorf("stating %s answered %d, want 400", body, status)
		}
	}
}

func TestAStatementOverTheBoundIsRefused(t *testing.T) {
	f := newFixture(t)
	alice := f.tokenFor("alice", "G/Alice/")

	rows := make([]string, MaxViewings+1)
	for i := range rows {
		rows[i] = `{"path":"G/Bob/screen","transport":"rtsp"}`
	}
	if status := f.state(alice, `{"viewing":[`+strings.Join(rows, ",")+`]}`); status != http.StatusBadRequest {
		t.Errorf("a statement of %d viewings answered %d, want 400", len(rows), status)
	}
}

func TestAReadNamesAPath(t *testing.T) {
	f := newFixture(t)
	if status, _ := f.viewersOf(f.tokenFor("bob", "G/Bob/"), ""); status != http.StatusBadRequest {
		t.Errorf("a read naming no path answered %d, want 400", status)
	}
}
