package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"task238-stagecue/internal/model"
	"task238-stagecue/internal/service"
	"task238-stagecue/internal/store"
)

func TestHTTPRoutesCreateRehearsalCueAndSelfCheck(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	server := httptest.NewServer(NewHandler(service.New(st)).Routes())
	defer server.Close()

	var rep model.Rehearsal
	status := postJSON(t, server.URL+"/api/rehearsals", map[string]string{"name": "http test", "show": "show"}, &rep)
	if status != http.StatusCreated || rep.ID == "" {
		t.Fatalf("create rehearsal status=%d body=%+v", status, rep)
	}

	var cue model.StageEvent
	status = postJSON(t, server.URL+"/api/cues", map[string]interface{}{
		"rehearsal_id": rep.ID, "seq": 1, "actor": string(model.ActorPerformer),
		"role": string(model.RoleCueStart), "label": "blackout", "raw_ts": int64(900), "device": "L1",
	}, &cue)
	if status != http.StatusCreated || cue.RehearsalID != rep.ID {
		t.Fatalf("create cue status=%d body=%+v", status, cue)
	}

	resp, err := http.Get(server.URL + "/api/rehearsals/" + rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get rehearsal status=%d", resp.StatusCode)
	}
	if _, err := io.ReadAll(resp.Body); err != nil {
		t.Fatal(err)
	}

	resp, err = http.Get(server.URL + "/api/selfcheck")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("selfcheck status=%d", resp.StatusCode)
	}
}

func postJSON(t *testing.T, url string, requestBody interface{}, responseBody interface{}) int {
	t.Helper()
	b, err := json.Marshal(requestBody)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil {
		t.Fatal(err)
	}
	return resp.StatusCode
}
