package store

import (
	"database/sql"
	"time"

	"task238-stagecue/internal/model"
)

// EventStore 舞台事件的持久化。
type EventStore struct{ s *Store }

func (s *Store) Events() *EventStore { return &EventStore{s: s} }

func (e *EventStore) Create(ev *model.StageEvent) error {
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	const q = `INSERT INTO stage_events(id,rehearsal_id,source,seq,actor,role,label,raw_timestamp,corrected_at,device,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?,?)`
	_, err := e.s.db.Exec(q, ev.ID, ev.RehearsalID, string(ev.Source), ev.Seq,
		string(ev.Actor), string(ev.Role), ev.Label, ev.RawTimestamp, ev.CorrectedAt,
		ev.Device, ev.CreatedAt.Format(time.RFC3339))
	return err
}

// UpsertBySeq 按 (rehearsal_id, seq) 幂等写入（seq>0 时幂等；seq<=0 直接插入）。
func (e *EventStore) UpsertBySeq(ev *model.StageEvent) error {
	if ev.CreatedAt.IsZero() {
		ev.CreatedAt = time.Now().UTC()
	}
	if ev.Seq <= 0 {
		return e.Create(ev)
	}
	// 先按 (rehearsal_id, seq) 查询已有记录，存在则更新，否则插入。
	const sel = `SELECT id FROM stage_events WHERE rehearsal_id=? AND seq=?`
	var existingID string
	err := e.s.db.QueryRow(sel, ev.RehearsalID, ev.Seq).Scan(&existingID)
	if err == sql.ErrNoRows {
		return e.Create(ev)
	}
	if err != nil {
		return err
	}
	const upd = `UPDATE stage_events SET source=?, actor=?, role=?, label=?, raw_timestamp=?, corrected_at=?, device=?, created_at=? WHERE id=?`
	_, err = e.s.db.Exec(upd, string(ev.Source), string(ev.Actor), string(ev.Role),
		ev.Label, ev.RawTimestamp, ev.CorrectedAt, ev.Device, ev.CreatedAt.Format(time.RFC3339), existingID)
	return err
}

func (e *EventStore) ListByRehearsal(rehearsalID string) ([]*model.StageEvent, error) {
	const q = `SELECT id,rehearsal_id,source,seq,actor,role,label,raw_timestamp,corrected_at,device,created_at
		FROM stage_events WHERE rehearsal_id=? ORDER BY corrected_at ASC, seq ASC`
	rows, err := e.s.db.Query(q, rehearsalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanEvents(rows)
}

// SetCorrectedAt 按事件 ID 回写校正后的统一时间戳（幂等，不触发插入冲突）。
func (e *EventStore) SetCorrectedAt(id string, correctedAt int64) error {
	const q = `UPDATE stage_events SET corrected_at=? WHERE id=?`
	_, err := e.s.db.Exec(q, correctedAt, id)
	return err
}

func (e *EventStore) Get(id string) (*model.StageEvent, error) {
	const q = `SELECT id,rehearsal_id,source,seq,actor,role,label,raw_timestamp,corrected_at,device,created_at
		FROM stage_events WHERE id=?`
	rows, err := e.s.db.Query(q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, model.ErrNotFound
	}
	ev, err := scanOneEvent(rows)
	if err != nil {
		return nil, err
	}
	return ev, rows.Err()
}

func scanEvents(rows *sql.Rows) ([]*model.StageEvent, error) {
	var out []*model.StageEvent
	for rows.Next() {
		ev, err := scanOneEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

func scanOneEvent(rows *sql.Rows) (*model.StageEvent, error) {
	var ev model.StageEvent
	var src, actor, role, ca string
	if err := rows.Scan(&ev.ID, &ev.RehearsalID, &src, &ev.Seq, &actor, &role,
		&ev.Label, &ev.RawTimestamp, &ev.CorrectedAt, &ev.Device, &ca); err != nil {
		return nil, err
	}
	ev.Source = model.SourceKind(src)
	ev.Actor = model.ActorKind(actor)
	ev.Role = model.EventRole(role)
	ev.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	return &ev, nil
}
