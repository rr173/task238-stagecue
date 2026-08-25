package rehearsal

import (
	"errors"
	"path/filepath"
	"testing"

	"task238-stagecue/internal/align"
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

// TestUpdateAnchorKeepsSingleRecord 锚点更新应原地改写同一条记录，
// 时间线只保留一条更新后的条目，且校正时间为 raw_ts - skew。
func TestUpdateAnchorKeepsSingleRecord(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	repSvc := New(st)
	alignSvc := align.New(st)
	rep, err := repSvc.Create("anchor test", "show")
	if err != nil {
		t.Fatal(err)
	}
	// cue_log 源时钟比统一基准慢 50ms。
	if err := alignSvc.SetSkew(rep.ID, model.SourceCueLog, 50, false); err != nil {
		t.Fatal(err)
	}
	ev, err := repSvc.AddEvent(rep.ID, model.SourceCueLog, 1, model.ActorPerformer,
		model.RoleCueStart, "blackout-start", 900, "L1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := alignSvc.Apply(rep.ID); err != nil {
		t.Fatal(err)
	}
	// 调整锚点：raw_ts=1200，应得校正时间 1200-50=1150。
	updated, err := repSvc.UpdateAnchor(ev.ID, 1200)
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != ev.ID {
		t.Fatalf("anchor update changed event id: got %s want %s", updated.ID, ev.ID)
	}
	if updated.CorrectedAt != 1150 {
		t.Fatalf("corrected time = %d, want 1150", updated.CorrectedAt)
	}
	timeline, err := alignSvc.Timeline(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline) != 1 {
		t.Fatalf("timeline length = %d, want 1 (anchor update duplicated the record)", len(timeline))
	}
	if timeline[0].CorrectedAt != 1150 || timeline[0].RawTimestamp != 1200 {
		t.Fatalf("timeline entry = {raw=%d, corrected=%d}, want {raw=1200, corrected=1150}",
			timeline[0].RawTimestamp, timeline[0].CorrectedAt)
	}
}
