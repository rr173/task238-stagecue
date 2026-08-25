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

func TestBug10_CrossRehearsalSupersedeIsRejectedAtomically(t *testing.T) {
	st,err:=store.Open(filepath.Join(t.TempDir(),"stagecue.db"));if err!=nil{t.Fatal(err)};defer st.Close();srv:=httptest.NewServer(NewHandler(service.New(st)).Routes());defer srv.Close();var rep1,rep2 model.Rehearsal;postBug10(t,srv.URL+"/api/rehearsals",map[string]any{"name":"one","show":"show"},&rep1);postBug10(t,srv.URL+"/api/rehearsals",map[string]any{"name":"two","show":"show"},&rep2);postBug10(t,srv.URL+"/api/rehearsals/"+rep1.ID+"/review",map[string]any{},&map[string]any{});postBug10(t,srv.URL+"/api/rehearsals/"+rep2.ID+"/review",map[string]any{},&map[string]any{});var oldPkg,newPkg model.CuePackage;postBug10(t,srv.URL+"/api/packages",map[string]any{"rehearsal_id":rep1.ID},&oldPkg);postBug10(t,srv.URL+"/api/packages/"+oldPkg.ID+"/publish",map[string]any{},&oldPkg);postBug10(t,srv.URL+"/api/packages",map[string]any{"rehearsal_id":rep2.ID},&newPkg);postBug10(t,srv.URL+"/api/packages/"+newPkg.ID+"/publish",map[string]any{},&newPkg)
	if s:=postBug10(t,srv.URL+"/api/packages/"+oldPkg.ID+"/supersede",map[string]any{"new_package_id":newPkg.ID},&map[string]any{});s!=http.StatusConflict{t.Fatalf("cross supersede status=%d",s)};var got model.CuePackage;r,err:=http.Get(srv.URL+"/api/packages/"+oldPkg.ID);if err!=nil{t.Fatal(err)};defer r.Body.Close();if err:=json.NewDecoder(r.Body).Decode(&got);err!=nil{t.Fatal(err)};if got.State!=model.StatePublished{t.Fatalf("old package state=%q",got.State)}
}
func postBug10(t *testing.T,url string,body,out any)int{t.Helper();b,_:=json.Marshal(body);r,err:=http.Post(url,"application/json",bytes.NewReader(b));if err!=nil{t.Fatal(err)};defer r.Body.Close();if err:=json.NewDecoder(r.Body).Decode(out);err!=nil{t.Fatal(err)};return r.StatusCode}
