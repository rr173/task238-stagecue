package cuepkg

import (
	"errors"
	"path/filepath"
	"testing"

	"task238-stagecue/internal/model"
	"task238-stagecue/internal/rehearsal"
	"task238-stagecue/internal/store"
)

func TestPublishFreezesSnapshotDigestAndReleasesRehearsal(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rep := &model.Rehearsal{ID: "rep_package_test", Name: "package", Show: "show", State: model.StatePending, CreatedAt: model.Now(), ImportedAt: model.Now()}
	if err := st.Rehearsals().Create(rep); err != nil {
		t.Fatal(err)
	}
	if err := st.Events().Create(&model.StageEvent{ID: "ev_package_test", RehearsalID: rep.ID, Source: model.SourceCueLog, Seq: 1, Actor: model.ActorPerformer, Role: model.RoleCueStart, Label: "cue", RawTimestamp: 1000, CreatedAt: model.Now()}); err != nil {
		t.Fatal(err)
	}

	svc := New(st)
	pkg, err := svc.Draft(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.SnapshotDigest == "" {
		t.Fatal("draft package has empty snapshot digest")
	}
	published, err := svc.Publish(pkg.ID)
	if err != nil {
		t.Fatal(err)
	}
	if published.State != model.StatePublished || published.SnapshotDigest != pkg.SnapshotDigest {
		t.Fatalf("published package = %+v", published)
	}
	updated, err := st.Rehearsals().Get(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.State != model.StateReviewable {
		t.Fatalf("rehearsal state after publish = %q, want %q", updated.State, model.StateReviewable)
	}
}

// TestSecondPublishFreezesRehearsalAndRejectsEventImport 锁定冻结边界：
// 当一个可发布（reviewable）演练版本再次发布时，演练转为 frozen（不可变快照），
// 之后任何场次事件导入必须被拒绝，而非继续写入。这覆盖发布路径触发的真实冻结，
// 而非直接调用 SetState(Frozen)，避免 store 层把 frozen 静默降级为 reviewable
// 从而绕过 AddEvent 的冻结守卫。
func TestSecondPublishFreezesRehearsalAndRejectsEventImport(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rep := &model.Rehearsal{
		ID: "rep_freeze_boundary", Name: "freeze", Show: "show",
		State: model.StatePending, CreatedAt: model.Now(), ImportedAt: model.Now(),
	}
	if err := st.Rehearsals().Create(rep); err != nil {
		t.Fatal(err)
	}
	if err := st.Events().Create(&model.StageEvent{
		ID: "ev_freeze_boundary", RehearsalID: rep.ID, Source: model.SourceCueLog,
		Seq: 1, Actor: model.ActorPerformer, Role: model.RoleCueStart,
		Label: "cue", RawTimestamp: 1000, CreatedAt: model.Now(),
	}); err != nil {
		t.Fatal(err)
	}

	pkgSvc := New(st)
	// 第一次发布：pending → reviewable。
	first, err := pkgSvc.Draft(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pkgSvc.Publish(first.ID); err != nil {
		t.Fatal(err)
	}
	// 第二次发布：reviewable → frozen（不可变边界）。
	second, err := pkgSvc.Draft(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pkgSvc.Publish(second.ID); err != nil {
		t.Fatalf("second publish (freeze): %v", err)
	}
	frozen, err := st.Rehearsals().Get(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if frozen.State != model.StateFrozen {
		t.Fatalf("rehearsal state after second publish = %q, want %q (frozen boundary must persist)",
			frozen.State, model.StateFrozen)
	}
	if frozen.FrozenAt.IsZero() {
		t.Fatal("frozen_at not set after freezing")
	}

	// 冻结后导入新事件必须被拒绝，而非成功写入。
	repSvc := rehearsal.New(st)
	_, err = repSvc.AddEvent(rep.ID, model.SourceScript, 2, model.ActorPerformer,
		model.RoleMoveOut, "late-import", 2000, "")
	if !errors.Is(err, model.ErrFrozenRehearsal) {
		t.Fatalf("post-freeze AddEvent error = %v, want %v", err, model.ErrFrozenRehearsal)
	}
	count, err := st.Rehearsals().CountEvents(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("event count after rejected import = %d, want 1 (frozen input must be immutable)", count)
	}
}
