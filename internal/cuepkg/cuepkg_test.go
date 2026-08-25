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
	// 发布后提示包必须处于已发布终态（而非被降级回草稿）。
	if published.State != model.StatePublished || published.SnapshotDigest != pkg.SnapshotDigest {
		t.Fatalf("published package = %+v", published)
	}
	// 发布即冻结演练输入，与演练状态机一致。
	updated, err := st.Rehearsals().Get(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.State != model.StateFrozen {
		t.Fatalf("rehearsal state after publish = %q, want %q", updated.State, model.StateFrozen)
	}

	// 冻结后事件导入与提示锚点修改都应被拒绝，保证已发布快照绑定的输入不可变。
	repSvc := rehearsal.New(st)
	if _, err := repSvc.AddEvent(rep.ID, model.SourceCueLog, 2, model.ActorPerformer,
		model.RoleCueStart, "late", 1100, "L2"); !errors.Is(err, model.ErrFrozenRehearsal) {
		t.Fatalf("post-publish AddEvent error = %v, want %v", err, model.ErrFrozenRehearsal)
	}
	if _, err := repSvc.UpdateAnchor("ev_package_test", 2000); !errors.Is(err, model.ErrFrozenRehearsal) {
		t.Fatalf("post-publish UpdateAnchor error = %v, want %v", err, model.ErrFrozenRehearsal)
	}

	// 重新发布同一提示包应被拒绝（已不在草稿/复核态）。
	if _, err := svc.Publish(pkg.ID); !errors.Is(err, model.ErrInvalidState) {
		t.Fatalf("re-publish error = %v, want %v", err, model.ErrInvalidState)
	}
}
