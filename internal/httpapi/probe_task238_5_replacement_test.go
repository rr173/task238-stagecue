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

func TestBug05_WaivedReviewAllowsDraftPackage(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := httptest.NewServer(NewHandler(service.New(st)).Routes())
	defer srv.Close()
	var rep model.Rehearsal
	postBug05New(t, srv.URL+"/api/rehearsals", map[string]any{"name": "waived-publish", "show": "show"}, &rep)
	for i, e := range []map[string]any{{"source": "cue_log", "seq": 1, "actor": "performer", "role": "cue_start", "raw_ts": 1000}, {"source": "cue_log", "seq": 2, "actor": "performer", "role": "cue_end", "raw_ts": 1500}, {"source": "device_log", "seq": 3, "actor": "performer", "role": "mech_move", "raw_ts": 1100}} {
		e["label"] = string(rune('a' + i))
		postBug05New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/events", e, &map[string]any{})
	}
	var ct model.Constraint
	postBug05New(t, srv.URL+"/api/constraints", map[string]any{"rehearsal_id": rep.ID, "name": "safe", "actor": "performer", "min_gap_ms": 0}, &ct)
	postBug05New(t, srv.URL+"/api/constraints/"+ct.ID+"/activate", map[string]any{}, &map[string]any{})
	var review service.ReviewResult
	postBug05New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/review", map[string]any{}, &review)
	if len(review.Conflicts) != 1 {
		t.Fatalf("review=%+v", review)
	}
	var waiver model.Waiver
	postBug05New(t, srv.URL+"/api/conflicts/"+review.Conflicts[0].ID+"/waiver", map[string]any{"reason": "covered", "evidence": "director"}, &waiver)
	postBug05New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/review", map[string]any{}, &review)
	if !review.Reviewable || review.Unresolved != 0 {
		t.Fatalf("review after waiver=%+v", review)
	}
	var pkg model.CuePackage
	if s := postBug05NewStatus(t, srv.URL+"/api/packages", map[string]any{"rehearsal_id": rep.ID}, &pkg); s != http.StatusCreated {
		t.Fatalf("draft status=%d body=%+v", s, pkg)
	}
}
func postBug05New(t *testing.T, url string, body, out any) {
	if s := postBug05NewStatus(t, url, body, out); s >= 300 {
		t.Fatalf("request %s status=%d", url, s)
	}
}
func postBug05NewStatus(t *testing.T, url string, body, out any) int {
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
