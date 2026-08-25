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

func TestBug09_RevokedConstraintStopsBlockingPublish(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := httptest.NewServer(NewHandler(service.New(st)).Routes())
	defer srv.Close()
	var rep model.Rehearsal
	postBug09New(t, srv.URL+"/api/rehearsals", map[string]any{"name": "revoke", "show": "show"}, &rep)
	for i, e := range []map[string]any{{"source": "cue_log", "seq": 1, "actor": "performer", "role": "cue_start", "raw_ts": 1000}, {"source": "cue_log", "seq": 2, "actor": "performer", "role": "cue_end", "raw_ts": 1500}, {"source": "device_log", "seq": 3, "actor": "performer", "role": "mech_move", "raw_ts": 1100}} {
		e["label"] = string(rune('a' + i))
		postBug09New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/events", e, &map[string]any{})
	}
	var ct model.Constraint
	postBug09New(t, srv.URL+"/api/constraints", map[string]any{"rehearsal_id": rep.ID, "name": "safe", "actor": "performer", "min_gap_ms": 0}, &ct)
	postBug09New(t, srv.URL+"/api/constraints/"+ct.ID+"/activate", map[string]any{}, &map[string]any{})
	var review service.ReviewResult
	postBug09New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/review", map[string]any{}, &review)
	if len(review.Conflicts) != 1 {
		t.Fatalf("initial review=%+v", review)
	}
	if s := postBug09NewStatus(t, srv.URL+"/api/constraints/"+ct.ID+"/revoke", map[string]any{}, &map[string]any{}); s != http.StatusOK {
		t.Fatalf("revoke status=%d", s)
	}
	postBug09New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/review", map[string]any{}, &review)
	if !review.Reviewable || review.Unresolved != 0 {
		t.Fatalf("review after revoke=%+v", review)
	}
	var conflicts []model.Conflict
	r, err := http.Get(srv.URL + "/api/rehearsals/" + rep.ID + "/conflicts")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&conflicts); err != nil {
		t.Fatal(err)
	}
	if len(conflicts) != 1 || !conflicts[0].Resolved {
		t.Fatalf("conflicts=%+v", conflicts)
	}
	var pkg model.CuePackage
	if s := postBug09NewStatus(t, srv.URL+"/api/packages", map[string]any{"rehearsal_id": rep.ID}, &pkg); s != http.StatusCreated {
		t.Fatalf("draft status=%d", s)
	}
}
func postBug09New(t *testing.T, url string, body, out any) {
	if s := postBug09NewStatus(t, url, body, out); s >= 300 {
		t.Fatalf("request %s status=%d", url, s)
	}
}
func postBug09NewStatus(t *testing.T, url string, body, out any) int {
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
