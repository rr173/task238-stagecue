package store

import (
	"path/filepath"
	"testing"

	"task238-stagecue/internal/model"
)

func TestOpenPersistsRehearsalAndEventsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stagecue.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	rep := &model.Rehearsal{
		ID: "rep_store_test", Name: "store test", Show: "show",
		State: model.StateImporting, CreatedAt: model.Now(), ImportedAt: model.Now(),
	}
	if err := st.Rehearsals().Create(rep); err != nil {
		t.Fatal(err)
	}
	if err := st.Events().Create(&model.StageEvent{
		ID: "event_store_test", RehearsalID: rep.ID, Source: model.SourceScript,
		Seq: 1, Actor: model.ActorPerformer, Role: model.RoleMoveOut,
		Label: "exit", RawTimestamp: 1000, CreatedAt: model.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.Rehearsals().Get(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != rep.Name || got.State != model.StateImporting {
		t.Fatalf("reopened rehearsal mismatch: %+v", got)
	}
	count, err := reopened.Rehearsals().CountEvents(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("event count after restart = %d, want 1", count)
	}
}
