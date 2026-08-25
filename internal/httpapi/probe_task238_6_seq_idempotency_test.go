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

func TestBug06_RepeatedCueSequenceKeepsOneEvent(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := httptest.NewServer(NewHandler(service.New(st)).Routes())
	defer srv.Close()
	var rep model.Rehearsal
	postBug06(t, srv.URL+"/api/rehearsals", map[string]any{"name": "idempotent", "show": "show"}, &rep)
	for _, label := range []string{"first", "retry"} {
		if s := postBug06(t, srv.URL+"/api/cues", map[string]any{"rehearsal_id": rep.ID, "seq": 9, "actor": "performer", "role": "cue_start", "raw_ts": 900, "label": label}, &map[string]any{}); s != http.StatusCreated {
			t.Fatalf("cue status=%d", s)
		}
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
	if len(timeline) != 1 || timeline[0].Label != "retry" {
		t.Fatalf("timeline=%+v, want one retry event", timeline)
	}
}
func postBug06(t *testing.T, url string, body, out any) int {
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
