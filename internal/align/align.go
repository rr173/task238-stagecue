package align

import (
	"sort"

	"task238-stagecue/internal/model"
	"task238-stagecue/internal/store"
)

// Service 对齐模块：设置各源时钟偏差、将事件原始时间戳校正到统一时钟、构建时间线。
type Service struct{ st *store.Store }

func New(st *store.Store) *Service { return &Service{st: st} }

// SetSkew 设置一个源的时钟偏差（毫秒），统一基准则为 0。
// 偏差变更后，重新校正该源已存在的事件并持久化，使后续读取一致。
func (s *Service) SetSkew(repID string, src model.SourceKind, skewMs int64, baseline bool) error {
	if _, err := s.st.Rehearsals().Get(repID); err != nil {
		return err
	}
	if err := s.st.Skews().Set(&model.ClockSkew{
		RehearsalID: repID, Source: src, SkewMs: skewMs, Baseline: baseline,
	}); err != nil {
		return err
	}
	return s.recorrectSource(repID, src, skewMs)
}

// recorrectSource 用新偏差重新校正某演练下指定源的所有事件并落库。
func (s *Service) recorrectSource(repID string, src model.SourceKind, skewMs int64) error {
	events, err := s.st.Events().ListByRehearsal(repID)
	if err != nil {
		return err
	}
	for _, ev := range events {
		if ev.Source != src {
			continue
		}
		corrected := ev.RawTimestamp - skewMs
		if ev.CorrectedAt == corrected {
			continue
		}
		if err := s.st.Events().SetCorrectedAt(ev.ID, corrected); err != nil {
			return err
		}
		ev.CorrectedAt = corrected
	}
	return nil
}

// Apply 校正某演练下所有事件的 corrected_at = raw_timestamp - skew_ms 并落库，
// 返回校正后的时间线条目（按统一时间升序）。
func (s *Service) Apply(repID string) ([]*model.StageEvent, error) {
	events, err := s.st.Events().ListByRehearsal(repID)
	if err != nil {
		return nil, err
	}
	skews, err := s.st.Skews().List(repID)
	if err != nil {
		return nil, err
	}
	skewOf := map[model.SourceKind]int64{}
	for _, sk := range skews {
		skewOf[sk.Source] = sk.SkewMs
	}
	for _, ev := range events {
		corrected := ev.RawTimestamp - skewOf[ev.Source]
		if ev.CorrectedAt == corrected {
			continue
		}
		if err := s.st.Events().SetCorrectedAt(ev.ID, corrected); err != nil {
			return nil, err
		}
		ev.CorrectedAt = corrected
	}
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].CorrectedAt == events[j].CorrectedAt {
			return events[i].Seq < events[j].Seq
		}
		return events[i].CorrectedAt < events[j].CorrectedAt
	})
	return events, nil
}

// Timeline 返回校正后的统一时间线（仅校正过 corrected_at 的事件）。
func (s *Service) Timeline(repID string) ([]*model.StageEvent, error) {
	return s.Apply(repID)
}

// Span 返回时间线起止（统一时钟毫秒）。
func (s *Service) Span(events []*model.StageEvent) (start, end int64) {
	if len(events) == 0 {
		return 0, 0
	}
	start = events[0].CorrectedAt
	end = events[len(events)-1].CorrectedAt
	for _, e := range events {
		if e.CorrectedAt < start {
			start = e.CorrectedAt
		}
		if e.CorrectedAt > end {
			end = e.CorrectedAt
		}
	}
	return start, end
}
