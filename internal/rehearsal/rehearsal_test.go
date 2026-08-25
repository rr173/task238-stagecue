package rehearsal

import (
	"errors"
	"path/filepath"
	"testing"

	"task238-stagecue/internal/model"
	"task238-stagecue/internal/store"
)

func TestAddEventIsIdempotentAndFrozenRehearsalRejectsWrites(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	svc := New(st)
	rep, err := svc.Create("rehearsal test", "show")
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"first", "replacement"} {
		if _, err := svc.AddEvent(rep.ID, model.SourceCueLog, 7, model.ActorPerformer,
			model.RoleCueStart, label, 900, "L1"); err != nil {
			t.Fatal(err)
		}
	}
	count, err := st.Rehearsals().CountEvents(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("idempotent event count = %d, want 1", count)
	}
	if err := st.Rehearsals().SetState(rep.ID, model.StateFrozen, model.Now()); err != nil {
		t.Fatal(err)
	}
	_, err = svc.AddEvent(rep.ID, model.SourceScript, 8, model.ActorPerformer,
		model.RoleMoveOut, "late", 1000, "")
	if !errors.Is(err, model.ErrFrozenRehearsal) {
		t.Fatalf("frozen write error = %v, want %v", err, model.ErrFrozenRehearsal)
	}
}
