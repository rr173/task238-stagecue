package rehearsal

import (
	"task238-stagecue/internal/model"
	"task238-stagecue/internal/store"
)

// Service 演练输入管理：创建演练版本并导入各源事件。
type Service struct{ st *store.Store }

func New(st *store.Store) *Service { return &Service{st: st} }

// Create 新建演练版本（导入中）。
func (s *Service) Create(name, show string) (*model.Rehearsal, error) {
	if name == "" {
		return nil, model.ErrInvalidState
	}
	rep := &model.Rehearsal{
		ID:         store.NewID("rep"),
		Name:       name,
		Show:       show,
		State:      model.StateImporting,
		CreatedAt:  model.Now(),
		ImportedAt: model.Now(),
	}
	if err := s.st.Rehearsals().Create(rep); err != nil {
		return nil, err
	}
	return rep, nil
}

// AddEvent 导入一条事件（按 seq 幂等）。
func (s *Service) AddEvent(repID string, src model.SourceKind, seq int, actor model.ActorKind,
	role model.EventRole, label string, rawTs int64, device string) (*model.StageEvent, error) {
	rep, err := s.st.Rehearsals().Get(repID)
	if err != nil {
		return nil, err
	}
	if rep.State == model.StateFrozen {
		return nil, model.ErrFrozenRehearsal
	}
	if rawTs <= 0 {
		return nil, model.ErrInvalidClockSkew
	}
	skew, err := s.st.Skews().SkewOf(repID, src)
	if err != nil {
		return nil, err
	}
	ev := &model.StageEvent{
		ID:           store.NewID("ev"),
		RehearsalID:  repID,
		Source:       src,
		Seq:          seq,
		Actor:        actor,
		Role:         role,
		Label:        label,
		RawTimestamp: rawTs,
		CorrectedAt:  rawTs - skew,
		Device:       device,
		CreatedAt:    model.Now(),
	}
	if err := s.st.Events().UpsertBySeq(ev); err != nil {
		return nil, err
	}
	if err := s.st.Packages().MarkDraftsStale(repID); err != nil {
		return nil, err
	}
	if err := s.st.Rehearsals().SetState(repID, model.StatePending, model.ZeroTime()); err != nil {
		return nil, err
	}
	return ev, nil
}

// Get 返回演练版本。
func (s *Service) Get(repID string) (*model.Rehearsal, error) {
	return s.st.Rehearsals().Get(repID)
}

// List 返回全部演练版本。
func (s *Service) List() ([]*model.Rehearsal, error) {
	return s.st.Rehearsals().List()
}

// MarkPending 将导入中转为待复核。
func (s *Service) MarkPending(repID string) error {
	rep, err := s.st.Rehearsals().Get(repID)
	if err != nil {
		return err
	}
	if rep.State != model.StateImporting {
		return model.ErrInvalidState
	}
	return s.st.Rehearsals().SetState(repID, model.StatePending, model.ZeroTime())
}

// UpdateAnchor changes a cue's source timestamp while preserving the
// rehearsal lifecycle invariant that frozen input is immutable.
func (s *Service) UpdateAnchor(eventID string, rawTimestamp int64) (*model.StageEvent, error) {
	ev, err := s.st.Events().Get(eventID)
	if err != nil {
		return nil, err
	}
	rep, err := s.st.Rehearsals().Get(ev.RehearsalID)
	if err != nil {
		return nil, err
	}
	if rep.State == model.StateFrozen {
		return nil, model.ErrFrozenRehearsal
	}
	if rawTimestamp <= 0 {
		return nil, model.ErrInvalidClockSkew
	}
	skew, err := s.st.Skews().SkewOf(ev.RehearsalID, ev.Source)
	if err != nil {
		return nil, err
	}
	ev.RawTimestamp = rawTimestamp
	ev.CorrectedAt = rawTimestamp - skew
	if err := s.st.Events().UpsertBySeq(ev); err != nil {
		return nil, err
	}
	return ev, nil
}
