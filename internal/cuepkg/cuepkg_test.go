package cuepkg

import (
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
