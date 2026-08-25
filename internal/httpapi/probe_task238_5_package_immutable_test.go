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

func TestBug05_PublishedPackageDigestRemainsStableAfterAnchorChange(t *testing.T) {
	st,err:=store.Open(filepath.Join(t.TempDir(),"stagecue.db"));if err!=nil{t.Fatal(err)};defer st.Close();srv:=httptest.NewServer(NewHandler(service.New(st)).Routes());defer srv.Close()
	var rep model.Rehearsal;postBug05(t,srv.URL+"/api/rehearsals",map[string]any{"name":"immutable","show":"show"},&rep);var ev model.StageEvent;postBug05(t,srv.URL+"/api/cues",map[string]any{"rehearsal_id":rep.ID,"seq":1,"actor":"performer","role":"cue_start","raw_ts":900,"label":"blackout"},&ev);postBug05(t,srv.URL+"/api/rehearsals/"+rep.ID+"/review",map[string]any{},&map[string]any{});var pkg model.CuePackage;postBug05(t,srv.URL+"/api/packages",map[string]any{"rehearsal_id":rep.ID},&pkg);postBug05(t,srv.URL+"/api/packages/"+pkg.ID+"/publish",map[string]any{},&pkg)
	if s:=patchBug05(t,srv.URL+"/api/cues/"+ev.ID+"/anchor",map[string]any{"raw_ts":1200},&map[string]any{});s!=http.StatusOK{t.Fatalf("anchor status=%d",s)}
	var got model.CuePackage;r,err:=http.Get(srv.URL+"/api/packages/"+pkg.ID);if err!=nil{t.Fatal(err)};defer r.Body.Close();if err:=json.NewDecoder(r.Body).Decode(&got);err!=nil{t.Fatal(err)};if got.State!=model.StatePublished||got.SnapshotDigest!=pkg.SnapshotDigest{t.Fatalf("package changed: before=%+v after=%+v",pkg,got)}
}
func postBug05(t *testing.T,url string,body,out any)int{t.Helper();b,_:=json.Marshal(body);r,err:=http.Post(url,"application/json",bytes.NewReader(b));if err!=nil{t.Fatal(err)};defer r.Body.Close();if err:=json.NewDecoder(r.Body).Decode(out);err!=nil{t.Fatal(err)};return r.StatusCode}
func patchBug05(t *testing.T,url string,body,out any)int{t.Helper();b,_:=json.Marshal(body);req,_:=http.NewRequest(http.MethodPatch,url,bytes.NewReader(b));req.Header.Set("Content-Type","application/json");r,err:=http.DefaultClient.Do(req);if err!=nil{t.Fatal(err)};defer r.Body.Close();if err:=json.NewDecoder(r.Body).Decode(out);err!=nil{t.Fatal(err)};return r.StatusCode}
