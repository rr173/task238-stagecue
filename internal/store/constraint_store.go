package store

import (
	"database/sql"
	"time"

	"task238-stagecue/internal/model"
)

// ConstraintStore 舞台约束的持久化。
type ConstraintStore struct{ s *Store }

func (s *Store) Constraints() *ConstraintStore { return &ConstraintStore{s: s} }

func (c *ConstraintStore) Create(ct *model.Constraint) error {
	if ct.CreatedAt.IsZero() {
		ct.CreatedAt = time.Now().UTC()
	}
	const q = `INSERT INTO constraints(id,rehearsal_id,name,actor,state,min_gap_ms,note,created_at)
		VALUES(?,?,?,?,?,?,?,?)`
	_, err := c.s.db.Exec(q, ct.ID, ct.RehearsalID, ct.Name, string(ct.Actor),
		string(ct.State), ct.MinGapMs, ct.Note, ct.CreatedAt.Format(time.RFC3339))
	return err
}

func (c *ConstraintStore) Get(id string) (*model.Constraint, error) {
	const q = `SELECT id,rehearsal_id,name,actor,state,min_gap_ms,note,created_at FROM constraints WHERE id=?`
	row := c.s.db.QueryRow(q, id)
	var ct model.Constraint
	var actor, ca string
	if err := row.Scan(&ct.ID, &ct.RehearsalID, &ct.Name, &actor, &ct.State,
		&ct.MinGapMs, &ct.Note, &ca); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	ct.Actor = model.ActorKind(actor)
	ct.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	return &ct, nil
}

func (c *ConstraintStore) ListByRehearsal(rehearsalID string) ([]*model.Constraint, error) {
	const q = `SELECT id,rehearsal_id,name,actor,state,min_gap_ms,note,created_at FROM constraints WHERE rehearsal_id=? ORDER BY created_at ASC`
	rows, err := c.s.db.Query(q, rehearsalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.Constraint
	for rows.Next() {
		var ct model.Constraint
		var actor, ca string
		if err := rows.Scan(&ct.ID, &ct.RehearsalID, &ct.Name, &actor, &ct.State,
			&ct.MinGapMs, &ct.Note, &ca); err != nil {
			return nil, err
		}
		ct.Actor = model.ActorKind(actor)
		ct.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		out = append(out, &ct)
	}
	return out, rows.Err()
}

func (c *ConstraintStore) SetState(id string, st model.State) error {
	const q = `UPDATE constraints SET state=? WHERE id=?`
	res, err := c.s.db.Exec(q, string(st), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (c *ConstraintStore) Revoke(id string) error {
	if err := c.SetState(id, model.StateRevoked); err != nil {
		return err
	}
	return c.s.Conflicts().ResolveByConstraint(id)
}
