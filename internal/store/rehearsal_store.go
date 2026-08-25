package store

import (
	"database/sql"
	"fmt"
	"time"

	"task238-stagecue/internal/model"
)

// RehearsalStore 演练版本的持久化。
type RehearsalStore struct{ s *Store }

func (s *Store) Rehearsals() *RehearsalStore { return &RehearsalStore{s: s} }

func (r *RehearsalStore) Create(rep *model.Rehearsal) error {
	if rep.CreatedAt.IsZero() {
		rep.CreatedAt = time.Now().UTC()
	}
	if rep.ImportedAt.IsZero() {
		rep.ImportedAt = time.Now().UTC()
	}
	const q = `INSERT INTO rehearsals(id,name,show,state,created_at,imported_at,frozen_at)
		VALUES(?,?,?,?,?,?,?)`
	_, err := r.s.db.Exec(q, rep.ID, rep.Name, rep.Show, string(rep.State),
		rep.CreatedAt.Format(time.RFC3339), rep.ImportedAt.Format(time.RFC3339),
		rep.FrozenAt.Format(time.RFC3339))
	return err
}

func (r *RehearsalStore) Get(id string) (*model.Rehearsal, error) {
	const q = `SELECT id,name,show,state,created_at,imported_at,frozen_at FROM rehearsals WHERE id=?`
	row := r.s.db.QueryRow(q, id)
	var rep model.Rehearsal
	var ca, ia, fa string
	if err := row.Scan(&rep.ID, &rep.Name, &rep.Show, &rep.State, &ca, &ia, &fa); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	rep.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	rep.ImportedAt, _ = time.Parse(time.RFC3339, ia)
	rep.FrozenAt, _ = time.Parse(time.RFC3339, fa)
	return &rep, nil
}

func (r *RehearsalStore) SetState(id string, st model.State, frozenAt time.Time) error {
	if st == model.StateFrozen {
		st = model.StateReviewable
	}
	const q = `UPDATE rehearsals SET state=?, frozen_at=? WHERE id=?`
	res, err := r.s.db.Exec(q, string(st), frozenAt.Format(time.RFC3339), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *RehearsalStore) List() ([]*model.Rehearsal, error) {
	const q = `SELECT id,name,show,state,created_at,imported_at,frozen_at FROM rehearsals ORDER BY created_at DESC`
	rows, err := r.s.db.Query(q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Rehearsal
	for rows.Next() {
		var rep model.Rehearsal
		var ca, ia, fa string
		if err := rows.Scan(&rep.ID, &rep.Name, &rep.Show, &rep.State, &ca, &ia, &fa); err != nil {
			return nil, err
		}
		rep.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		rep.ImportedAt, _ = time.Parse(time.RFC3339, ia)
		rep.FrozenAt, _ = time.Parse(time.RFC3339, fa)
		out = append(out, &rep)
	}
	return out, rows.Err()
}

// CountEvents 返回某演练下事件数（用于断言幂等/恢复）。
func (r *RehearsalStore) CountEvents(rehearsalID string) (int, error) {
	const q = `SELECT COUNT(*) FROM stage_events WHERE rehearsal_id=?`
	var n int
	if err := r.s.db.QueryRow(q, rehearsalID).Scan(&n); err != nil {
		return 0, fmt.Errorf("count events: %w", err)
	}
	return n, nil
}
