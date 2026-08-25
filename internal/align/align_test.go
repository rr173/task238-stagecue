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

// TestApplyPersistsCorrectedAtAcrossReads 锁定持久化契约：配置时钟偏差后，
// 时间线刷新与单事件重新读取都必须返回已校正的统一时间戳，而非原始时间戳。
func TestApplyPersistsCorrectedAtAcrossReads(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	repSvc := rehearsal.New(st)
	rep, err := repSvc.Create("persist test", "show")
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
	// 触发一次复核刷新，将校正结果落库。
	if _, err := svc.Apply(rep.ID); err != nil {
		t.Fatal(err)
	}
	// 时间线刷新必须返回已校正时间戳 950。
	timeline, err := st.Events().ListByRehearsal(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline) != 1 || timeline[0].CorrectedAt != 950 {
		t.Fatalf("timeline corrected_at = %d, want 950", timeline[0].CorrectedAt)
	}
	// 单事件重新读取必须同样返回已校正时间戳 950（持久化生效，而非仅内存态）。
	ev, err := st.Events().Get(timeline[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if ev.CorrectedAt != 950 {
		t.Fatalf("re-read corrected_at = %d, want 950", ev.CorrectedAt)
	}
	if ev.RawTimestamp != 1000 {
		t.Fatalf("raw_timestamp = %d, want 1000", ev.RawTimestamp)
	}
}
