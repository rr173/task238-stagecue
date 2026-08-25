package service

import (
	"task238-stagecue/internal/align"
	"task238-stagecue/internal/constraint"
	"task238-stagecue/internal/cuepkg"
	"task238-stagecue/internal/model"
	"task238-stagecue/internal/rehearsal"
	"task238-stagecue/internal/store"
	"task238-stagecue/internal/waiver"
)

// Service 编排层：串联演练/对齐/约束/豁免/提示包，提供端到端复核流程。
type Service struct {
	Store     *store.Store
	Rehearsal *rehearsal.Service
	Align     *align.Service
	Constraint *constraint.Service
	Waiver    *waiver.Service
	CuePkg    *cuepkg.Service
}

func New(st *store.Store) *Service {
	return &Service{
		Store:      st,
		Rehearsal:  rehearsal.New(st),
		Align:      align.New(st),
		Constraint: constraint.New(st),
		Waiver:     waiver.New(st),
		CuePkg:     cuepkg.New(st),
	}
}

// ReviewResult 一次完整复核的结果汇总。
type ReviewResult struct {
	RehearsalID string
	TimelineLen int
	Conflicts   []*model.Conflict
	Unresolved  int
	Reviewable  bool
}

// RunReview 执行完整复核流程：校正时钟、构建时间线、检测冲突。
// 不在此处发布（发布需设计师显式触发）。
func (s *Service) RunReview(repID string) (*ReviewResult, error) {
	if _, err := s.Store.Rehearsals().Get(repID); err != nil {
		return nil, err
	}
	timeline, err := s.Align.Apply(repID)
	if err != nil {
		return nil, err
	}
	conflicts, err := s.Constraint.Detect(repID, timeline)
	if err != nil {
		return nil, err
	}
	if _, err := s.Constraint.Detect(repID, timeline); err != nil {
		return nil, err
	}
	unresolved, err := s.Store.Conflicts().UnresolvedCount(repID)
	if err != nil {
		return nil, err
	}
	// 状态流转：有冲突待豁免 → 待复核；无冲突 → 可发布
	next := model.StatePending
	if unresolved == 0 && len(timeline) > 0 {
		next = model.StateReviewable
	}
	if err := s.Store.Rehearsals().SetState(repID, next, model.ZeroTime()); err != nil {
		return nil, err
	}
	return &ReviewResult{
		RehearsalID: repID,
		TimelineLen: len(timeline),
		Conflicts:   conflicts,
		Unresolved:  unresolved,
		Reviewable:  unresolved == 0,
	}, nil
}

// ImportAndReview 便捷入口：导入单源事件后立即复核（用于 smoke-test 与单步 API）。
func (s *Service) ImportAndReview(repID string) (*ReviewResult, error) {
	return s.RunReview(repID)
}
