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

func TestBug04_TimelineRefreshPersistsCorrectedTimestamp(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := httptest.NewServer(NewHandler(service.New(st)).Routes())
	defer srv.Close()
	var rep model.Rehearsal
	postBug04New(t, srv.URL+"/api/rehearsals", map[string]any{"name": "persist-skew", "show": "show"}, &rep)
	postBug04New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/skew", map[string]any{"source": "device_log", "skew_ms": 50}, &map[string]any{})
	var ev model.StageEvent
	postBug04New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/events", map[string]any{"source": "device_log", "seq": 1, "actor": "performer", "role": "mech_move", "raw_ts": 1000}, &ev)
	var timeline []model.StageEvent
	r, err := http.Get(srv.URL + "/api/rehearsals/" + rep.ID + "/timeline")
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Fatalf("timeline status=%d", r.StatusCode)
	}
	if err := json.NewDecoder(r.Body).Decode(&timeline); err != nil {
		t.Fatal(err)
	}
	if len(timeline) != 1 || timeline[0].CorrectedAt != 950 {
		t.Fatalf("timeline=%+v", timeline)
	}
	r2, err := http.Get(srv.URL + "/api/cues/" + ev.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer r2.Body.Close()
	var got model.StageEvent
	if err := json.NewDecoder(r2.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.CorrectedAt != 950 {
		t.Fatalf("stored corrected_at=%d", got.CorrectedAt)
	}
}
func postBug04New(t *testing.T, url string, body, out any) {
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
