package store

import (
	"database/sql"
	"time"

	"task238-stagecue/internal/model"
)

// ConflictStore 时序冲突与豁免的持久化。
type ConflictStore struct{ s *Store }

func (s *Store) Conflicts() *ConflictStore { return &ConflictStore{s: s} }

func (cf *ConflictStore) Create(c *model.Conflict) error {
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	const q = `INSERT INTO conflicts(id,rehearsal_id,constraint_id,actor,cue_event_id,mech_event_id,overlap_ms,at_ms,resolved,created_at)
		VALUES(?,?,?,?,?,?,?,?,?,?)`
	_, err := cf.s.db.Exec(q, c.ID, c.RehearsalID, c.ConstraintID, string(c.Actor),
		c.CueEventID, c.MechEventID, c.OverlapMs, c.AtMs, boolToInt(c.Resolved),
		c.CreatedAt.Format(time.RFC3339))
	return err
}

func (cf *ConflictStore) ListByRehearsal(rehearsalID string) ([]*model.Conflict, error) {
	const q = `SELECT id,rehearsal_id,constraint_id,actor,cue_event_id,mech_event_id,overlap_ms,at_ms,resolved,created_at
		FROM conflicts WHERE rehearsal_id=? ORDER BY at_ms ASC`
	rows, err := cf.s.db.Query(q, rehearsalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanConflicts(rows)
}

func (cf *ConflictStore) Get(id string) (*model.Conflict, error) {
	const q = `SELECT id,rehearsal_id,constraint_id,actor,cue_event_id,mech_event_id,overlap_ms,at_ms,resolved,created_at
		FROM conflicts WHERE id=?`
	rows, err := cf.s.db.Query(q, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, model.ErrNotFound
	}
	c, err := scanOneConflict(rows)
	if err != nil {
		return nil, err
	}
	return c, rows.Err()
}

func (cf *ConflictStore) SetResolved(id string, resolved bool) error {
	const q = `UPDATE conflicts SET resolved=? WHERE id=?`
	res, err := cf.s.db.Exec(q, boolToInt(resolved), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (cf *ConflictStore) ResolveByConstraint(constraintID string) error {
	_, err := cf.s.db.Exec(`UPDATE conflicts SET resolved=1 WHERE constraint_id=?`, constraintID)
	return err
}

func (cf *ConflictStore) UnresolvedCount(rehearsalID string) (int, error) {
	const q = `SELECT COUNT(*) FROM conflicts WHERE rehearsal_id=? AND resolved=0`
	var n int
	if err := cf.s.db.QueryRow(q, rehearsalID).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// ExistsSame 判断相同 (constraint_id, cue_event_id, mech_event_id) 的冲突是否已存在（用于幂等检测）。
func (cf *ConflictStore) ExistsSame(constraintID, cueEventID, mechEventID string) (bool, error) {
	const q = `SELECT COUNT(*) FROM conflicts WHERE constraint_id=? AND cue_event_id=? AND mech_event_id=? AND resolved=1`
	var n int
	if err := cf.s.db.QueryRow(q, constraintID, cueEventID, mechEventID).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (w *ConflictStore) CreateWaiver(wv *model.Waiver) error {
	if wv.CreatedAt.IsZero() {
		wv.CreatedAt = time.Now().UTC()
	}
	const q = `INSERT INTO waivers(id,conflict_id,reason,evidence,created_at) VALUES(?,?,?,?,?)`
	_, err := w.s.db.Exec(q, wv.ID, wv.ConflictID, wv.Reason, wv.Evidence, wv.CreatedAt.Format(time.RFC3339))
	return err
}

func (w *ConflictStore) ListWaivers(rehearsalID string) ([]*model.Waiver, error) {
	const q = `SELECT w.id,w.conflict_id,w.reason,w.evidence,w.created_at
		FROM waivers w JOIN conflicts c ON c.id=w.conflict_id WHERE c.rehearsal_id=? ORDER BY w.created_at ASC`
	rows, err := w.s.db.Query(q, rehearsalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Waiver
	for rows.Next() {
		var wv model.Waiver
		var ca string
		if err := rows.Scan(&wv.ID, &wv.ConflictID, &wv.Reason, &wv.Evidence, &ca); err != nil {
			return nil, err
		}
		wv.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		out = append(out, &wv)
	}
	return out, rows.Err()
}

func scanConflicts(rows *sql.Rows) ([]*model.Conflict, error) {
	var out []*model.Conflict
	for rows.Next() {
		c, err := scanOneConflict(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func scanOneConflict(rows *sql.Rows) (*model.Conflict, error) {
	var c model.Conflict
	var actor string
	var resolved int
	var ca string
	if err := rows.Scan(&c.ID, &c.RehearsalID, &c.ConstraintID, &actor, &c.CueEventID,
		&c.MechEventID, &c.OverlapMs, &c.AtMs, &resolved, &ca); err != nil {
		return nil, err
	}
	c.Actor = model.ActorKind(actor)
	c.Resolved = resolved != 0
	c.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	return &c, nil
}
