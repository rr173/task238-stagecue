package waiver

import (
	"task238-stagecue/internal/model"
	"task238-stagecue/internal/store"
)

// Service 豁免模块：对检测到的冲突登记豁免理由并标记解决。
type Service struct{ st *store.Store }

func New(st *store.Store) *Service { return &Service{st: st} }

// Register 为某冲突登记豁免证据；登记后该冲突标记为已解决。
func (s *Service) Register(conflictID, reason, evidence string) (*model.Waiver, error) {
	cf, err := s.st.Conflicts().Get(conflictID)
	if err != nil {
		return nil, err
	}
	if cf.Resolved {
		return nil, model.ErrConflict
	}
	if reason == "" {
		return nil, model.ErrMissingWaiver
	}
	wv := &model.Waiver{
		ID:         store.NewID("wv"),
		ConflictID: conflictID,
		Reason:     reason,
		Evidence:   evidence,
		CreatedAt:  model.Now(),
	}
	if err := s.st.Conflicts().CreateWaiver(wv); err != nil {
		return nil, err
	}
	if err := s.st.Conflicts().SetResolved(conflictID, true); err != nil {
		return nil, err
	}
	return wv, nil
}

// List 返回某演练下全部豁免。
func (s *Service) List(repID string) ([]*model.Waiver, error) {
	return s.st.Conflicts().ListWaivers(repID)
}

// ConflictResolution 返回某冲突是否已有豁免。
func (s *Service) ConflictResolution(conflictID string) (resolved bool, reason string, err error) {
	cf, err := s.st.Conflicts().Get(conflictID)
	if err != nil {
		return false, "", err
	}
	if !cf.Resolved {
		return false, "", nil
	}
	wvs, err := s.st.Conflicts().ListWaivers(cf.RehearsalID)
	if err != nil {
		return true, "", nil
	}
	for _, wv := range wvs {
		if wv.ConflictID == conflictID {
			return true, wv.Reason, nil
		}
	}
	return true, "", nil
}
