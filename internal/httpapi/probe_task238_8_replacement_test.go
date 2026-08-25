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

func TestBug08_FrozenLifecycleSurvivesRestart(t *testing.T) {
	db := filepath.Join(t.TempDir(), "stagecue.db")
	st, err := store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewHandler(service.New(st)).Routes())
	var rep model.Rehearsal
	postBug08New(t, srv.URL+"/api/rehearsals", map[string]any{"name": "restart-freeze", "show": "show"}, &rep)
	var cue model.StageEvent
	postBug08New(t, srv.URL+"/api/cues", map[string]any{"rehearsal_id": rep.ID, "seq": 1, "actor": "performer", "role": "cue_start", "raw_ts": 900}, &cue)
	if s := postBug08NewStatus(t, srv.URL+"/api/rehearsals/"+rep.ID+"/freeze", map[string]any{}, &map[string]any{}); s != http.StatusOK {
		t.Fatalf("freeze status=%d", s)
	}
	srv.Close()
	st.Close()
	st2, err := store.Open(db)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	srv2 := httptest.NewServer(NewHandler(service.New(st2)).Routes())
	defer srv2.Close()
	if s := postBug08NewStatus(t, srv2.URL+"/api/rehearsals/"+rep.ID+"/events", map[string]any{"source": "script", "seq": 2, "actor": "performer", "role": "move_out", "raw_ts": 1200}, &map[string]any{}); s != http.StatusConflict {
		t.Fatalf("event status=%d", s)
	}
	if s := patchBug08New(t, srv2.URL+"/api/cues/"+cue.ID+"/anchor", map[string]any{"raw_ts": 1000}, &map[string]any{}); s != http.StatusConflict {
		t.Fatalf("anchor status=%d", s)
	}
}
func postBug08New(t *testing.T, url string, body, out any) {
	if s := postBug08NewStatus(t, url, body, out); s >= 300 {
		t.Fatalf("request status=%d", s)
	}
}
func postBug08NewStatus(t *testing.T, url string, body, out any) int {
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
func patchBug08New(t *testing.T, url string, body, out any) int {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPatch, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	r, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		t.Fatal(err)
	}
	return r.StatusCode
}
