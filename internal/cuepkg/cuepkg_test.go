package cuepkg

import (
	"errors"
	"path/filepath"
	"testing"

	"task238-stagecue/internal/model"
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

// TestPublishRejectsStaleDraftAfterNewEvent 复现：
// 复核后创建草稿提示包，再新增一条演练事件，旧草稿发布应被拒绝（快照失效），
// 且草稿记录仍保持草稿状态。
func TestPublishRejectsStaleDraftAfterNewEvent(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "stagecue.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rep := &model.Rehearsal{ID: "rep_stale_test", Name: "stale", Show: "show", State: model.StatePending, CreatedAt: model.Now(), ImportedAt: model.Now()}
	if err := st.Rehearsals().Create(rep); err != nil {
		t.Fatal(err)
	}
	if err := st.Events().Create(&model.StageEvent{ID: "ev_stale_1", RehearsalID: rep.ID, Source: model.SourceCueLog, Seq: 1, Actor: model.ActorPerformer, Role: model.RoleCueStart, Label: "cue", RawTimestamp: 1000, CreatedAt: model.Now()}); err != nil {
		t.Fatal(err)
	}

	svc := New(st)
	// 复核后创建草稿提示包（固化当前演练输入的快照摘要）。
	draft, err := svc.Draft(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	originalDigest := draft.SnapshotDigest

	// 再新增一条演练事件：演练输入变更，旧草稿的快照摘要失效。
	if err := st.Events().Create(&model.StageEvent{ID: "ev_stale_2", RehearsalID: rep.ID, Source: model.SourceCueLog, Seq: 2, Actor: model.ActorPerformer, Role: model.RoleCueEnd, Label: "cue-end", RawTimestamp: 1200, CreatedAt: model.Now()}); err != nil {
		t.Fatal(err)
	}

	// 发布旧草稿应被拒绝（快照失效）。
	if _, err := svc.Publish(draft.ID); !errors.Is(err, model.ErrStaleSnapshot) {
		t.Fatalf("publish stale draft error = %v, want %v", err, model.ErrStaleSnapshot)
	}
	// 草稿记录仍保持草稿状态（未被发布，也未被废弃）。
	after, err := svc.Get(draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.State != model.StateDraft {
		t.Fatalf("stale draft state = %q, want %q", after.State, model.StateDraft)
	}
	if after.SnapshotDigest != originalDigest {
		t.Fatalf("stale draft digest changed = %q, want %q", after.SnapshotDigest, originalDigest)
	}
	// 演练不应被推进为可发布/冻结。
	updatedRep, err := st.Rehearsals().Get(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updatedRep.State == model.StateReviewable || updatedRep.State == model.StateFrozen {
		t.Fatalf("rehearsal advanced to %q despite stale publish", updatedRep.State)
	}

	// 复核后重新创建草稿即可正常发布（新草稿反映最新演练输入）。
	if err := st.Rehearsals().SetState(rep.ID, model.StatePending, model.ZeroTime()); err != nil {
		t.Fatal(err)
	}
	fresh, err := svc.Draft(rep.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.SnapshotDigest == originalDigest {
		t.Fatal("fresh draft digest matches stale digest; expected changed snapshot")
	}
	published, err := svc.Publish(fresh.ID)
	if err != nil {
		t.Fatalf("publish fresh draft: %v", err)
	}
	if published.State != model.StatePublished {
		t.Fatalf("fresh published state = %q, want %q", published.State, model.StatePublished)
	}
	// 旧草稿依然保持草稿状态，未被新草稿影响。
	stillStale, err := svc.Get(draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stillStale.State != model.StateDraft {
		t.Fatalf("old stale draft state = %q, want %q", stillStale.State, model.StateDraft)
	}
}
