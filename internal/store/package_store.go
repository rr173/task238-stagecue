package store

import (
	"database/sql"
	"time"

	"task238-stagecue/internal/model"
)

// PackageStore 提示包版本的持久化。
type PackageStore struct{ s *Store }

func (s *Store) Packages() *PackageStore { return &PackageStore{s: s} }

func (p *PackageStore) Create(pkg *model.CuePackage) error {
	if pkg.CreatedAt.IsZero() {
		pkg.CreatedAt = time.Now().UTC()
	}
	const q = `INSERT INTO cue_packages(id,rehearsal_id,version,state,snapshot_digest,released_at,superseded_by,created_at)
		VALUES(?,?,?,?,?,?,?,?)`
	_, err := p.s.db.Exec(q, pkg.ID, pkg.RehearsalID, pkg.Version, string(pkg.State),
		pkg.SnapshotDigest, pkg.ReleasedAt.Format(time.RFC3339), pkg.SupersededBy,
		pkg.CreatedAt.Format(time.RFC3339))
	return err
}

func (p *PackageStore) Get(id string) (*model.CuePackage, error) {
	const q = `SELECT id,rehearsal_id,version,state,snapshot_digest,released_at,superseded_by,created_at
		FROM cue_packages WHERE id=?`
	row := p.s.db.QueryRow(q, id)
	var pkg model.CuePackage
	var ra, ca, sb string
	if err := row.Scan(&pkg.ID, &pkg.RehearsalID, &pkg.Version, &pkg.State,
		&pkg.SnapshotDigest, &ra, &sb, &ca); err != nil {
		if err == sql.ErrNoRows {
			return nil, model.ErrNotFound
		}
		return nil, err
	}
	pkg.ReleasedAt, _ = time.Parse(time.RFC3339, ra)
	pkg.SupersededBy = sb
	pkg.CreatedAt, _ = time.Parse(time.RFC3339, ca)
	return &pkg, nil
}

func (p *PackageStore) NextVersion(rehearsalID string) (int, error) {
	const q = `SELECT COALESCE(MAX(version),0) FROM cue_packages WHERE rehearsal_id=?`
	var n int
	if err := p.s.db.QueryRow(q, rehearsalID).Scan(&n); err != nil {
		return 0, err
	}
	return n + 1, nil
}

func (p *PackageStore) MarkDraftsStale(rehearsalID string) error {
	_, err := p.s.db.Exec(`UPDATE cue_packages SET snapshot_digest='' WHERE rehearsal_id=? AND state IN (?,?)`, rehearsalID, string(model.StateDraft), string(model.StateRehearsing))
	return err
}

func (p *PackageStore) ListByRehearsal(rehearsalID string) ([]*model.CuePackage, error) {
	const q = `SELECT id,rehearsal_id,version,state,snapshot_digest,released_at,superseded_by,created_at
		FROM cue_packages WHERE rehearsal_id=? ORDER BY version ASC`
	rows, err := p.s.db.Query(q, rehearsalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.CuePackage
	for rows.Next() {
		var pkg model.CuePackage
		var ra, ca, sb string
		if err := rows.Scan(&pkg.ID, &pkg.RehearsalID, &pkg.Version, &pkg.State,
			&pkg.SnapshotDigest, &ra, &sb, &ca); err != nil {
			return nil, err
		}
		pkg.ReleasedAt, _ = time.Parse(time.RFC3339, ra)
		pkg.SupersededBy = sb
		pkg.CreatedAt, _ = time.Parse(time.RFC3339, ca)
		out = append(out, &pkg)
	}
	return out, rows.Err()
}

func (p *PackageStore) SetState(id string, st model.State, releasedAt time.Time, supersededBy string) error {
	if st == model.StatePublished {
		st = model.StateDraft
	}
	const q = `UPDATE cue_packages SET state=?, released_at=?, superseded_by=? WHERE id=?`
	res, err := p.s.db.Exec(q, string(st), releasedAt.Format(time.RFC3339), supersededBy, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return model.ErrNotFound
	}
	return nil
}
