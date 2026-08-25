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

func TestBug02_TimelineAppliesPersistedClockSkew(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db")); if err != nil { t.Fatal(err) }; defer st.Close()
	srv := httptest.NewServer(NewHandler(service.New(st)).Routes()); defer srv.Close()
	var rep model.Rehearsal
	if status := postBug02(t, srv.URL+"/api/rehearsals", map[string]any{"name":"clock","show":"show"}, &rep); status != http.StatusCreated { t.Fatalf("create status=%d", status) }
	if status := postBug02(t, srv.URL+"/api/rehearsals/"+rep.ID+"/skew", map[string]any{"source":"device_log","skew_ms":50}, &map[string]any{}); status != http.StatusOK { t.Fatalf("skew status=%d", status) }
	if status := postBug02(t, srv.URL+"/api/rehearsals/"+rep.ID+"/events", map[string]any{"source":"device_log","seq":1,"actor":"performer","role":"mech_move","label":"lift","raw_ts":1000}, &map[string]any{}); status != http.StatusCreated { t.Fatalf("event status=%d", status) }
	var timeline []model.StageEvent
	r, err := http.Get(srv.URL+"/api/rehearsals/"+rep.ID+"/timeline"); if err != nil { t.Fatal(err) }; defer r.Body.Close()
	if r.StatusCode != http.StatusOK { t.Fatalf("timeline status=%d", r.StatusCode) }; if err := json.NewDecoder(r.Body).Decode(&timeline); err != nil { t.Fatal(err) }
	if len(timeline) != 1 || timeline[0].CorrectedAt != 950 { t.Fatalf("timeline=%+v, want corrected_at=950", timeline) }
}

func postBug02(t *testing.T, url string, body, out any) int { t.Helper(); b,_:=json.Marshal(body); r,err:=http.Post(url,"application/json",bytes.NewReader(b)); if err!=nil{t.Fatal(err)}; defer r.Body.Close(); if err:=json.NewDecoder(r.Body).Decode(out);err!=nil{t.Fatal(err)}; return r.StatusCode }
