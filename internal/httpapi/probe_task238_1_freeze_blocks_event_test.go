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

func TestBug01_FrozenRehearsalRejectsEventAfterFreeze(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil { t.Fatal(err) }
	defer st.Close()
	srv := httptest.NewServer(NewHandler(service.New(st)).Routes())
	defer srv.Close()
	var rep model.Rehearsal
	if status := postBug01(t, srv.URL+"/api/rehearsals", map[string]any{"name":"freeze","show":"show"}, &rep); status != http.StatusCreated { t.Fatalf("create status=%d", status) }
	if status := postBug01(t, srv.URL+"/api/rehearsals/"+rep.ID+"/freeze", map[string]any{}, &map[string]any{}); status != http.StatusOK { t.Fatalf("freeze status=%d", status) }
	status := postBug01(t, srv.URL+"/api/rehearsals/"+rep.ID+"/events", map[string]any{"source":"script","seq":1,"actor":"performer","role":"move_out","label":"exit","raw_ts":1000}, &map[string]any{})
	if status != http.StatusConflict { t.Fatalf("frozen event status=%d, want %d", status, http.StatusConflict) }
}

func postBug01(t *testing.T, url string, body, out any) int {
	t.Helper(); b, _ := json.Marshal(body)
	r, err := http.Post(url, "application/json", bytes.NewReader(b)); if err != nil { t.Fatal(err) }; defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(out); err != nil { t.Fatal(err) }; return r.StatusCode
}
