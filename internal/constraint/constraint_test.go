package constraint

import (
	"path/filepath"
	"testing"

	"task238-stagecue/internal/align"
	"task238-stagecue/internal/model"
	"task238-stagecue/internal/rehearsal"
	"task238-stagecue/internal/store"
)

func TestDetectPersistsOneConflictAndIsIdempotent(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	repSvc := rehearsal.New(st)
	rep, err := repSvc.Create("constraint test", "show")
	if err != nil {
		t.Fatal(err)
	}
	for seq, event := range []struct {
		role model.EventRole
		ts   int64
	}{
		{model.RoleCueStart, 1000}, {model.RoleCueEnd, 1500}, {model.RoleMechMove, 1100},
	} {
		if _, err := repSvc.AddEvent(rep.ID, model.SourceCueLog, seq+1, model.ActorPerformer,
			event.role, "event", event.ts, "device"); err != nil {
			t.Fatal(err)
		}
	}
	svc := New(st)
	ct, err := svc.Add(rep.ID, "safe gap", model.ActorPerformer, 0, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.Activate(ct.ID); err != nil {
		t.Fatal(err)
	}
	timeline, err := align.New(st).Apply(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	first, err := svc.Detect(rep.ID, timeline)
	if err != nil {
		t.Fatal(err)
	}
	second, err := svc.Detect(rep.ID, timeline)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 1 || len(second) != 0 {
		t.Fatalf("detect results = (%d,%d), want (1,0)", len(first), len(second))
	}
	stored, err := st.Conflicts().ListByRehearsal(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 1 || stored[0].OverlapMs != 400 {
		t.Fatalf("stored conflicts = %+v", stored)
	}
}
