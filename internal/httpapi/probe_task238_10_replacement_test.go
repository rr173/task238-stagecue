package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"task238-stagecue/internal/model"
	"task238-stagecue/internal/service"
	"task238-stagecue/internal/store"
	"testing"
)

func TestBug10_NewEventInvalidatesDraftPackage(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := httptest.NewServer(NewHandler(service.New(st)).Routes())
	defer srv.Close()
	var rep model.Rehearsal
	postBug10New(t, srv.URL+"/api/rehearsals", map[string]any{"name": "stale-draft", "show": "show"}, &rep)
	postBug10New(t, srv.URL+"/api/cues", map[string]any{"rehearsal_id": rep.ID, "seq": 1, "actor": "performer", "role": "cue_start", "raw_ts": 900}, &map[string]any{})
	postBug10New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/review", map[string]any{}, &map[string]any{})
	var pkg model.CuePackage
	if s := postBug10NewStatus(t, srv.URL+"/api/packages", map[string]any{"rehearsal_id": rep.ID}, &pkg); s != http.StatusCreated {
		t.Fatalf("draft status=%d", s)
	}
	postBug10New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/events", map[string]any{"source": "script", "seq": 2, "actor": "performer", "role": "move_out", "raw_ts": 1200}, &map[string]any{})
	if s := postBug10NewStatus(t, srv.URL+"/api/packages/"+pkg.ID+"/publish", map[string]any{}, &map[string]any{}); s != http.StatusConflict {
		t.Fatalf("publish status=%d", s)
	}
	var got model.CuePackage
	r, err := http.Get(srv.URL + "/api/packages/" + pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.State != model.StateDraft {
		t.Fatalf("package=%+v", got)
	}
}
func postBug10New(t *testing.T, url string, body, out any) {
	if s := postBug10NewStatus(t, url, body, out); s >= 300 {
		t.Fatalf("request %s status=%d", url, s)
	}
}
func postBug10NewStatus(t *testing.T, url string, body, out any) int {
	t.Helper()
	b, _ := json.Marshal(body)
	r, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		t.Fatal(err)
	}
	return r.StatusCode
}
