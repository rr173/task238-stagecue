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

func TestBug07_InvalidSupersedeLeavesPublishedPackageUntouched(t *testing.T) {
	st,err:=store.Open(filepath.Join(t.TempDir(),"stagecue.db"));if err!=nil{t.Fatal(err)};defer st.Close();srv:=httptest.NewServer(NewHandler(service.New(st)).Routes());defer srv.Close();var rep model.Rehearsal;postBug07(t,srv.URL+"/api/rehearsals",map[string]any{"name":"atomic","show":"show"},&rep);postBug07(t,srv.URL+"/api/rehearsals/"+rep.ID+"/review",map[string]any{},&map[string]any{});var pkg model.CuePackage;postBug07(t,srv.URL+"/api/packages",map[string]any{"rehearsal_id":rep.ID},&pkg);postBug07(t,srv.URL+"/api/packages/"+pkg.ID+"/publish",map[string]any{},&pkg)
	if s:=postBug07(t,srv.URL+"/api/packages/"+pkg.ID+"/supersede",map[string]any{"new_package_id":"missing"},&map[string]any{});s!=http.StatusNotFound{t.Fatalf("supersede status=%d",s)}
	var got model.CuePackage;r,err:=http.Get(srv.URL+"/api/packages/"+pkg.ID);if err!=nil{t.Fatal(err)};defer r.Body.Close();if err:=json.NewDecoder(r.Body).Decode(&got);err!=nil{t.Fatal(err)};if got.State!=model.StatePublished||got.SupersededBy!=""{t.Fatalf("package corrupted=%+v",got)}
}
func postBug07(t *testing.T,url string,body,out any)int{t.Helper();b,_:=json.Marshal(body);r,err:=http.Post(url,"application/json",bytes.NewReader(b));if err!=nil{t.Fatal(err)};defer r.Body.Close();if err:=json.NewDecoder(r.Body).Decode(out);err!=nil{t.Fatal(err)};return r.StatusCode}
