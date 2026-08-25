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

func TestBug04_PublishKeepsPackageDraftWhenConflictIsUnresolved(t *testing.T) {
	st,err:=store.Open(filepath.Join(t.TempDir(),"stagecue.db"));if err!=nil{t.Fatal(err)};defer st.Close();srv:=httptest.NewServer(NewHandler(service.New(st)).Routes());defer srv.Close()
	var rep model.Rehearsal;postBug04(t,srv.URL+"/api/rehearsals",map[string]any{"name":"publish","show":"show"},&rep)
	for i,e:=range []map[string]any{{"source":"cue_log","seq":1,"actor":"performer","role":"cue_start","raw_ts":1000},{"source":"cue_log","seq":2,"actor":"performer","role":"cue_end","raw_ts":1500},{"source":"device_log","seq":3,"actor":"performer","role":"mech_move","raw_ts":1100}}{e["label"]=string(rune('a'+i));if s:=postBug04(t,srv.URL+"/api/rehearsals/"+rep.ID+"/events",e,&map[string]any{});s!=http.StatusCreated{t.Fatalf("event status=%d",s)}}
	var ct model.Constraint;postBug04(t,srv.URL+"/api/constraints",map[string]any{"rehearsal_id":rep.ID,"name":"safe","actor":"performer","min_gap_ms":0},&ct);postBug04(t,srv.URL+"/api/constraints/"+ct.ID+"/activate",map[string]any{},&map[string]any{});postBug04(t,srv.URL+"/api/rehearsals/"+rep.ID+"/review",map[string]any{},&map[string]any{})
	var pkg model.CuePackage;if s:=postBug04(t,srv.URL+"/api/packages",map[string]any{"rehearsal_id":rep.ID},&pkg);s!=http.StatusCreated{t.Fatalf("draft status=%d",s)}
	if s:=postBug04(t,srv.URL+"/api/packages/"+pkg.ID+"/publish",map[string]any{},&map[string]any{});s!=http.StatusConflict{t.Fatalf("publish status=%d",s)}
	var got model.CuePackage;r,err:=http.Get(srv.URL+"/api/packages/"+pkg.ID);if err!=nil{t.Fatal(err)};defer r.Body.Close();if err:=json.NewDecoder(r.Body).Decode(&got);err!=nil{t.Fatal(err)};if got.State!=model.StateDraft{t.Fatalf("package state=%q, want draft",got.State)}
}
func postBug04(t *testing.T,url string,body,out any)int{t.Helper();b,_:=json.Marshal(body);r,err:=http.Post(url,"application/json",bytes.NewReader(b));if err!=nil{t.Fatal(err)};defer r.Body.Close();if err:=json.NewDecoder(r.Body).Decode(out);err!=nil{t.Fatal(err)};return r.StatusCode}
