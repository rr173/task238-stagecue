package waiver

import (
	"path/filepath"
	"testing"

	"task238-stagecue/internal/model"
	"task238-stagecue/internal/store"
)

func TestRegisterResolvesConflictAndPreservesEvidence(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rep := &model.Rehearsal{ID: "rep_waiver_test", Name: "waiver", Show: "show", State: model.StatePending, CreatedAt: model.Now(), ImportedAt: model.Now()}
	if err := st.Rehearsals().Create(rep); err != nil {
		t.Fatal(err)
	}
	ct := &model.Constraint{ID: "con_waiver_test", RehearsalID: rep.ID, Name: "safe", Actor: model.ActorPerformer, State: model.StateActive, CreatedAt: model.Now()}
	if err := st.Constraints().Create(ct); err != nil {
		t.Fatal(err)
	}
	cf := &model.Conflict{ID: "cf_waiver_test", RehearsalID: rep.ID, ConstraintID: ct.ID, Actor: model.ActorPerformer, CueEventID: "cue", MechEventID: "mech", OverlapMs: 50, AtMs: 1000, CreatedAt: model.Now()}
	if err := st.Conflicts().Create(cf); err != nil {
		t.Fatal(err)
	}

	wv, err := New(st).Register(cf.ID, "artistic blackout", "director approval")
	if err != nil {
		t.Fatal(err)
	}
	if wv.Reason != "artistic blackout" || wv.Evidence != "director approval" {
		t.Fatalf("waiver = %+v", wv)
	}
	resolved, reason, err := New(st).ConflictResolution(cf.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !resolved || reason != wv.Reason {
		t.Fatalf("resolution = (%v,%q), want (true,%q)", resolved, reason, wv.Reason)
	}
}
