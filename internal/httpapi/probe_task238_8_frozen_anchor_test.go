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

func TestBug08_FrozenRehearsalRejectsCueAnchorChange(t *testing.T) {
	st,err:=store.Open(filepath.Join(t.TempDir(),"stagecue.db"));if err!=nil{t.Fatal(err)};defer st.Close();srv:=httptest.NewServer(NewHandler(service.New(st)).Routes());defer srv.Close();var rep model.Rehearsal;postBug08(t,srv.URL+"/api/rehearsals",map[string]any{"name":"anchor","show":"show"},&rep);var ev model.StageEvent;postBug08(t,srv.URL+"/api/cues",map[string]any{"rehearsal_id":rep.ID,"seq":1,"actor":"performer","role":"cue_start","raw_ts":900},&ev);postBug08(t,srv.URL+"/api/rehearsals/"+rep.ID+"/freeze",map[string]any{},&map[string]any{});if s:=patchBug08(t,srv.URL+"/api/cues/"+ev.ID+"/anchor",map[string]any{"raw_ts":1000},&map[string]any{});s!=http.StatusConflict{t.Fatalf("anchor status=%d, want %d",s,http.StatusConflict)}
}
func postBug08(t *testing.T,url string,body,out any)int{t.Helper();b,_:=json.Marshal(body);r,err:=http.Post(url,"application/json",bytes.NewReader(b));if err!=nil{t.Fatal(err)};defer r.Body.Close();if err:=json.NewDecoder(r.Body).Decode(out);err!=nil{t.Fatal(err)};return r.StatusCode}
func patchBug08(t *testing.T,url string,body,out any)int{t.Helper();b,_:=json.Marshal(body);req,_:=http.NewRequest(http.MethodPatch,url,bytes.NewReader(b));req.Header.Set("Content-Type","application/json");r,err:=http.DefaultClient.Do(req);if err!=nil{t.Fatal(err)};defer r.Body.Close();if err:=json.NewDecoder(r.Body).Decode(out);err!=nil{t.Fatal(err)};return r.StatusCode}
