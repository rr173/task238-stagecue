package rehearsal

import (
	"errors"
	"path/filepath"
	"testing"

	"task238-stagecue/internal/cuepkg"
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

// TestFreezeSurvivesRestart verifies the durable freeze boundary: after the
// process closes and reopens the same SQLite file, frozen input stays
// immutable. Both new event writes (场次事件) and cue anchor edits (灯光提示
// 锚点) are rejected on both sides of the restart — the freeze must not be
// forgotten across the reopen.
func TestFreezeSurvivesRestart(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stagecue.db")

	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st)
	rep, err := svc.Create("restart freeze", "show")
	if err != nil {
		t.Fatal(err)
	}
	cue, err := svc.AddEvent(rep.ID, model.SourceCueLog, 1, model.ActorPerformer,
		model.RoleCueStart, "blackout", 900, "L1")
	if err != nil {
		t.Fatal(err)
	}
	// Freeze via the publish path (reviewable → frozen, frozen_at recorded).
	if err := st.Rehearsals().SetState(rep.ID, model.StateReviewable, model.ZeroTime()); err != nil {
		t.Fatal(err)
	}
	cuesvc := cuepkg.New(st)
	pkg, err := cuesvc.Draft(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cuesvc.Publish(pkg.ID); err != nil {
		t.Fatal(err)
	}
	// Sanity: the freeze marker is now durable in the store.
	repFrozen, err := st.Rehearsals().Get(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if repFrozen.FrozenAt.IsZero() {
		t.Fatal("publish did not record a frozen_at marker")
	}

	// Pre-restart: both kinds of frozen edits are rejected.
	if _, err := svc.AddEvent(rep.ID, model.SourceScript, 2, model.ActorPerformer,
		model.RoleMoveOut, "late", 1000, ""); !errors.Is(err, model.ErrFrozenRehearsal) {
		t.Fatalf("pre-restart frozen addEvent error = %v, want %v", err, model.ErrFrozenRehearsal)
	}
	if _, err := svc.UpdateAnchor(cue.ID, 950); !errors.Is(err, model.ErrFrozenRehearsal) {
		t.Fatalf("pre-restart frozen updateAnchor error = %v, want %v", err, model.ErrFrozenRehearsal)
	}

	// Restart: close and reopen the same database file.
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st2, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st2.Close()
	svc2 := New(st2)

	// Post-restart: the freeze boundary must persist — both edits still rejected.
	if _, err := svc2.AddEvent(rep.ID, model.SourceScript, 3, model.ActorPerformer,
		model.RoleMoveOut, "post-restart", 1000, ""); !errors.Is(err, model.ErrFrozenRehearsal) {
		t.Fatalf("post-restart frozen addEvent error = %v, want %v", err, model.ErrFrozenRehearsal)
	}
	if _, err := svc2.UpdateAnchor(cue.ID, 950); !errors.Is(err, model.ErrFrozenRehearsal) {
		t.Fatalf("post-restart frozen updateAnchor error = %v, want %v", err, model.ErrFrozenRehearsal)
	}
}
