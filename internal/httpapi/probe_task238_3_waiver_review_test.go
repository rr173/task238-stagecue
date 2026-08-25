package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task238-stagecue/internal/model"
	"task238-stagecue/internal/service"
	"task238-stagecue/internal/store"
)

func TestBug03_WaivedConflictStaysResolvedAfterReview(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := httptest.NewServer(NewHandler(service.New(st)).Routes())
	defer srv.Close()
	var rep model.Rehearsal
	postBug03(t, srv.URL+"/api/rehearsals", map[string]any{"name": "waiver", "show": "show"}, &rep)
	for i, e := range []map[string]any{{"source": "cue_log", "seq": 1, "actor": "performer", "role": "cue_start", "raw_ts": 1000}, {"source": "cue_log", "seq": 2, "actor": "performer", "role": "cue_end", "raw_ts": 1500}, {"source": "device_log", "seq": 3, "actor": "performer", "role": "mech_move", "raw_ts": 1100}} {
		e["label"] = string(rune('a' + i))
		if s := postBug03(t, srv.URL+"/api/rehearsals/"+rep.ID+"/events", e, &map[string]any{}); s != http.StatusCreated {
			t.Fatalf("event status=%d", s)
		}
	}
	var constraint model.Constraint
	postBug03(t, srv.URL+"/api/constraints", map[string]any{"rehearsal_id": rep.ID, "name": "safe", "actor": "performer", "min_gap_ms": 0}, &constraint)
	if s := postBug03(t, srv.URL+"/api/constraints/"+constraint.ID+"/activate", map[string]any{}, &map[string]any{}); s != http.StatusOK {
		t.Fatalf("activate status=%d", s)
	}
	var review service.ReviewResult
	postBug03(t, srv.URL+"/api/rehearsals/"+rep.ID+"/review", map[string]any{}, &review)
	if len(review.Conflicts) != 1 {
		t.Fatalf("review=%+v", review)
	}
	var waiver model.Waiver
	if s := postBug03(t, srv.URL+"/api/conflicts/"+review.Conflicts[0].ID+"/waiver", map[string]any{"reason": "covered", "evidence": "director"}, &waiver); s != http.StatusCreated {
		t.Fatalf("waiver status=%d", s)
	}
	postBug03(t, srv.URL+"/api/rehearsals/"+rep.ID+"/review", map[string]any{}, &review)
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
		t.Fatalf("conflicts=%+v, want one resolved conflict", conflicts)
	}
}
func postBug03(t *testing.T, url string, body, out any) int {
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
