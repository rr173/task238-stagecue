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

func TestBug07_AnchorUpdateIsIdempotentAndNormalized(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := httptest.NewServer(NewHandler(service.New(st)).Routes())
	defer srv.Close()
	var rep model.Rehearsal
	postBug07New(t, srv.URL+"/api/rehearsals", map[string]any{"name": "anchor-update", "show": "show"}, &rep)
	postBug07New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/skew", map[string]any{"source": "cue_log", "skew_ms": 50}, &map[string]any{})
	var ev model.StageEvent
	postBug07New(t, srv.URL+"/api/cues", map[string]any{"rehearsal_id": rep.ID, "seq": 1, "actor": "performer", "role": "cue_start", "raw_ts": 900}, &ev)
	var updated model.StageEvent
	var updatedBody map[string]any
	if s := patchBug07New(t, srv.URL+"/api/cues/"+ev.ID+"/anchor", map[string]any{"raw_ts": 1200}, &updatedBody); s != http.StatusOK {
		t.Fatalf("anchor status=%d body=%+v", s, updatedBody)
	}
	b, _ := json.Marshal(updatedBody)
	_ = json.Unmarshal(b, &updated)
	if updated.CorrectedAt != 1150 {
		t.Fatalf("response=%+v", updated)
	}
	var timeline []model.StageEvent
	r, err := http.Get(srv.URL + "/api/rehearsals/" + rep.ID + "/timeline")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(&timeline); err != nil {
		t.Fatal(err)
	}
	if len(timeline) != 1 || timeline[0].RawTimestamp != 1200 || timeline[0].CorrectedAt != 1150 {
		t.Fatalf("timeline=%+v", timeline)
	}
}
func postBug07New(t *testing.T, url string, body, out any) {
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
}
func patchBug07New(t *testing.T, url string, body, out any) int {
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
