package constraint

import (
	"time"

	"task238-stagecue/internal/model"
	"task238-stagecue/internal/store"
)

// Service 约束模块：维护舞台约束并检测互斥照明/机械动作的时序冲突。
type Service struct{ st *store.Store }

func New(st *store.Store) *Service { return &Service{st: st} }

// span 表示一个时间区间 [start, end] 及关联事件 ID。
type span struct {
	start, end int64
	eventID    string
}

// Add 新增约束（草稿）。
func (s *Service) Add(repID, name string, actor model.ActorKind, minGapMs int64, note string) (*model.Constraint, error) {
	if _, err := s.st.Rehearsals().Get(repID); err != nil {
		return nil, err
	}
	if minGapMs < 0 {
		return nil, model.ErrNegativeDuration
	}
	ct := &model.Constraint{
		ID:          store.NewID("con"),
		RehearsalID: repID,
		Name:        name,
		Actor:       actor,
		State:       model.StateDraft,
		MinGapMs:    minGapMs,
		Note:        note,
		CreatedAt:   nowUTC(),
	}
	if err := s.st.Constraints().Create(ct); err != nil {
		return nil, err
	}
	return ct, nil
}

// Activate 将约束从草稿转为生效。
func (s *Service) Activate(id string) error {
	ct, err := s.st.Constraints().Get(id)
	if err != nil {
		return err
	}
	if ct.State != model.StateDraft {
		return model.ErrInvalidState
	}
	return s.st.Constraints().SetState(id, model.StateActive)
}

// Revoke 废止约束。
func (s *Service) Revoke(id string) error {
	ct, err := s.st.Constraints().Get(id)
	if err != nil {
		return err
	}
	if ct.State == model.StateRevoked {
		return model.ErrInvalidState
	}
	return s.st.Constraints().SetState(id, model.StateRevoked)
}

// List 返回演练下全部约束。
func (s *Service) List(repID string) ([]*model.Constraint, error) {
	return s.st.Constraints().ListByRehearsal(repID)
}

// Detect 在统一时间线上检测冲突：对每条生效约束，
// 比较同一 actor 下照明提示区间(cue_start..cue_end)与机械动作占用区间(mech_move 起占用 mechOccupancyMs)
// 是否重叠超过 min_gap_ms 容差。返回新检测到的冲突（已落库）。
func (s *Service) Detect(repID string, timeline []*model.StageEvent) ([]*model.Conflict, error) {
	constraints, err := s.st.Constraints().ListByRehearsal(repID)
	if err != nil {
		return nil, err
	}
	// 构建每个 actor 的照明区间与机械动作占用区间。
	cueSpans := map[model.ActorKind][]span{}
	mechSpans := map[model.ActorKind][]span{}
	for _, ev := range timeline {
		if ev.Role == model.RoleCueStart {
			cueSpans[ev.Actor] = append(cueSpans[ev.Actor], span{ev.CorrectedAt, ev.CorrectedAt, ev.ID})
		} else if ev.Role == model.RoleCueEnd {
			// 将 cue_end 闭合最近一个未闭合的 cue_start 区间，保留 cue_start 的 ID。
			if spans := cueSpans[ev.Actor]; len(spans) > 0 {
				last := &spans[len(spans)-1]
				if ev.CorrectedAt > last.end {
					last.end = ev.CorrectedAt
				}
			}
		} else if ev.Role == model.RoleMechMove {
			mechSpans[ev.Actor] = append(mechSpans[ev.Actor],
				span{ev.CorrectedAt, ev.CorrectedAt + mechOccupancyMs, ev.ID})
		}
	}

	var conflicts []*model.Conflict
	for _, ct := range constraints {
		if ct.State != model.StateActive {
			continue
		}
		for _, cs := range cueSpans[ct.Actor] {
			for _, ms := range mechSpans[ct.Actor] {
				overlap := overlapMs(cs, ms)
				if overlap < 0 {
					continue // 不重叠
				}
				if overlap >= ct.MinGapMs {
					// 幂等：相同 (约束, 照明事件, 机械事件) 已检测过则跳过，保留既有 resolved 状态。
					if ok, e := s.st.Conflicts().ExistsSame(ct.ID, cs.eventID+":"+string(ct.Actor), ms.eventID); e != nil {
						return nil, e
					} else if ok {
						continue
					}
					c := &model.Conflict{
						ID:           store.NewID("cf"),
						RehearsalID:  repID,
						ConstraintID: ct.ID,
						Actor:        ct.Actor,
						CueEventID:   cs.eventID,
						MechEventID:  ms.eventID,
						OverlapMs:    overlap,
						AtMs:         max64(cs.start, ms.start),
						CreatedAt:    nowUTC(),
					}
					if err := s.st.Conflicts().Create(c); err != nil {
						return nil, err
					}
					conflicts = append(conflicts, c)
				}
			}
		}
	}
	return conflicts, nil
}

// mechOccupancyMs 机械动作从触发起的占用（危险）窗口毫秒。
const mechOccupancyMs int64 = 400

// overlapMs 返回两个区间的重叠毫秒；不重叠返回 -1。
func overlapMs(a, b span) int64 {
	s := max64(a.start, b.start)
	e := min64(a.end, b.end)
	if e <= s {
		return -1
	}
	return e - s
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func min64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func nowUTC() time.Time { return model.Now() }
