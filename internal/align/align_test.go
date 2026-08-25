package align

import (
	"path/filepath"
	"testing"

	"task238-stagecue/internal/model"
	"task238-stagecue/internal/rehearsal"
	"task238-stagecue/internal/store"
)

func TestApplyNormalizesSourceClockAndOrdersTimeline(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	repSvc := rehearsal.New(st)
	rep, err := repSvc.Create("alignment test", "show")
	if err != nil {
		t.Fatal(err)
	}
	svc := New(st)
	if err := svc.SetSkew(rep.ID, model.SourceDeviceLog, 50, false); err != nil {
		t.Fatal(err)
	}
	if _, err := repSvc.AddEvent(rep.ID, model.SourceDeviceLog, 1, model.ActorPerformer,
		model.RoleMechMove, "mechanism", 1000, "M1"); err != nil {
		t.Fatal(err)
	}
	if _, err := repSvc.AddEvent(rep.ID, model.SourceScript, 2, model.ActorPerformer,
		model.RoleMoveOut, "exit", 900, ""); err != nil {
		t.Fatal(err)
	}
	events, err := svc.Apply(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("timeline length = %d, want 2", len(events))
	}
	if events[0].CorrectedAt != 900 || events[1].CorrectedAt != 950 {
		t.Fatalf("corrected timeline = [%d,%d], want [900,950]", events[0].CorrectedAt, events[1].CorrectedAt)
	}
}
