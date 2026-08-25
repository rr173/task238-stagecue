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

func TestBug02_PublishedPackageFreezesInputWrites(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	srv := httptest.NewServer(NewHandler(service.New(st)).Routes())
	defer srv.Close()
	var rep model.Rehearsal
	postBug02New(t, srv.URL+"/api/rehearsals", map[string]any{"name": "publish-freeze", "show": "show"}, &rep)
	var cue model.StageEvent
	postBug02New(t, srv.URL+"/api/cues", map[string]any{"rehearsal_id": rep.ID, "seq": 1, "actor": "performer", "role": "cue_start", "raw_ts": 900}, &cue)
	postBug02New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/review", map[string]any{}, &map[string]any{})
	var pkg model.CuePackage
	if s := postBug02New(t, srv.URL+"/api/packages", map[string]any{"rehearsal_id": rep.ID}, &pkg); s != http.StatusCreated {
		t.Fatalf("draft status=%d", s)
	}
	if s := postBug02New(t, srv.URL+"/api/packages/"+pkg.ID+"/publish", map[string]any{}, &pkg); s != http.StatusOK {
		t.Fatalf("publish status=%d", s)
	}
	var published model.CuePackage
	if s := getBug02New(t, srv.URL+"/api/packages/"+pkg.ID, &published); s != http.StatusOK || published.State != model.StatePublished {
		t.Fatalf("package after publish status=%d package=%+v", s, published)
	}
	if s := postBug02New(t, srv.URL+"/api/rehearsals/"+rep.ID+"/events", map[string]any{"source": "script", "seq": 2, "actor": "performer", "role": "move_out", "raw_ts": 1200}, &map[string]any{}); s != http.StatusConflict {
		t.Fatalf("event status=%d", s)
	}
	if s := patchBug02New(t, srv.URL+"/api/cues/"+cue.ID+"/anchor", map[string]any{"raw_ts": 1000}, &map[string]any{}); s != http.StatusConflict {
		t.Fatalf("anchor status=%d", s)
	}
}
func postBug02New(t *testing.T, url string, body, out any) int {
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
func patchBug02New(t *testing.T, url string, body, out any) int {
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
func getBug02New(t *testing.T, url string, out any) int {
	t.Helper()
	r, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil {
		t.Fatal(err)
	}
	return r.StatusCode
}
