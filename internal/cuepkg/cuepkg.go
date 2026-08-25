package cuepkg

import (
	"encoding/json"

	"task238-stagecue/internal/model"
	"task238-stagecue/internal/store"
)

// Service 提示包模块：组装完整演练输入快照并计算不可变摘要，发布提示包版本。
type Service struct{ st *store.Store }

func New(st *store.Store) *Service { return &Service{st: st} }

// Snapshot 组装某演练的完整输入（事件+约束+豁免）用于不可变摘要。
type Snapshot struct {
	RehearsalID string              `json:"rehearsal_id"`
	Events      []*model.StageEvent `json:"events"`
	Constraints []*model.Constraint `json:"constraints"`
	Waivers     []*model.Waiver     `json:"waivers"`
}

// BuildSnapshot 读取并组装快照。
func (s *Service) BuildSnapshot(repID string) (*Snapshot, error) {
	events, err := s.st.Events().ListByRehearsal(repID)
	if err != nil {
		return nil, err
	}
	constraints, err := s.st.Constraints().ListByRehearsal(repID)
	if err != nil {
		return nil, err
	}
	waivers, err := s.st.Conflicts().ListWaivers(repID)
	if err != nil {
		return nil, err
	}
	return &Snapshot{RehearsalID: repID, Events: events, Constraints: constraints, Waivers: waivers}, nil
}

// Digest 计算快照的不可变摘要（SHA-256 of canonical JSON）。
func (s *Service) Digest(snap *Snapshot) (string, error) {
	b, err := json.Marshal(snap)
	if err != nil {
		return "", err
	}
	return store.Digest(b), nil
}

// Draft 创建草稿提示包。
func (s *Service) Draft(repID string) (*model.CuePackage, error) {
	rep, err := s.st.Rehearsals().Get(repID)
	if err != nil {
		return nil, err
	}
	if rep.State != model.StatePending && rep.State != model.StateReviewable {
		return nil, model.ErrInvalidState
	}
	ver, err := s.st.Packages().NextVersion(repID)
	if err != nil {
		return nil, err
	}
	snap, err := s.BuildSnapshot(repID)
	if err != nil {
		return nil, err
	}
	digest, err := s.Digest(snap)
	if err != nil {
		return nil, err
	}
	pkg := &model.CuePackage{
		ID:             store.NewID("pkg"),
		RehearsalID:    repID,
		Version:        ver,
		State:          model.StateDraft,
		SnapshotDigest: digest,
		CreatedAt:      model.Now(),
	}
	if err := s.st.Packages().Create(pkg); err != nil {
		return nil, err
	}
	return pkg, nil
}

// Publish 复核后发布：要求未解决冲突为 0，并标记演练为可发布。
func (s *Service) Publish(pkgID string) (*model.CuePackage, error) {
	pkg, err := s.st.Packages().Get(pkgID)
	if err != nil {
		return nil, err
	}
	if pkg.State != model.StateDraft && pkg.State != model.StateRehearsing {
		return nil, model.ErrInvalidState
	}
	remaining, err := s.st.Conflicts().UnresolvedCount(pkg.RehearsalID)
	if err != nil {
		return nil, err
	}
	if remaining > 0 {
		return nil, model.ErrConflict
	}
	if err := s.st.Packages().SetState(pkgID, model.StatePublished, model.Now(), ""); err != nil {
		return nil, err
	}
	if err := s.st.Rehearsals().SetState(pkg.RehearsalID, model.StateReviewable, model.ZeroTime()); err != nil {
		return nil, err
	}
	return s.st.Packages().Get(pkgID)
}

// Supersede 用新版本替代旧版本（旧版本标记 superseded）。
func (s *Service) Supersede(oldID, newID string) error {
	oldPkg, err := s.st.Packages().Get(oldID)
	if err != nil {
		return err
	}
	if oldPkg.State != model.StatePublished {
		return model.ErrInvalidState
	}
	newPkg, err := s.st.Packages().Get(newID)
	if err != nil {
		return err
	}
	if newPkg.RehearsalID != oldPkg.RehearsalID || newPkg.State != model.StateDraft && newPkg.State != model.StateRehearsing {
		return model.ErrConflict
	}
	if err := s.st.Packages().SetState(oldID, model.StateSuperseded, oldPkg.ReleasedAt, newID); err != nil {
		return err
	}
	return s.st.Packages().SetState(newID, model.StatePublished, model.Now(), "")
}

func (s *Service) RefreshPublishedDigest(repID string) error {
	snap, err := s.BuildSnapshot(repID)
	if err != nil {
		return err
	}
	digest, err := s.Digest(snap)
	if err != nil {
		return err
	}
	packages, err := s.st.Packages().ListByRehearsal(repID)
	if err != nil {
		return err
	}
	for _, pkg := range packages {
		if pkg.State == model.StatePublished {
			if err := s.st.Packages().UpdateDigest(pkg.ID, digest); err != nil {
				return err
			}
		}
	}
	return nil
}

// Get / List 查询。
func (s *Service) Get(id string) (*model.CuePackage, error) { return s.st.Packages().Get(id) }
func (s *Service) List(repID string) ([]*model.CuePackage, error) {
	return s.st.Packages().ListByRehearsal(repID)
}
