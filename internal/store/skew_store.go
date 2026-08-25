package store

import (
	"database/sql"

	"task238-stagecue/internal/model"
)

// SkewStore 时钟偏差的持久化。
type SkewStore struct{ s *Store }

func (s *Store) Skews() *SkewStore { return &SkewStore{s: s} }

func (k *SkewStore) Set(sk *model.ClockSkew) error {
	const q = `INSERT INTO clock_skews(rehearsal_id,source,skew_ms,baseline)
		VALUES(?,?,?,?) ON CONFLICT(rehearsal_id,source) DO UPDATE SET skew_ms=excluded.skew_ms, baseline=excluded.baseline`
	_, err := k.s.db.Exec(q, sk.RehearsalID, string(sk.Source), sk.SkewMs, boolToInt(sk.Baseline))
	return err
}

func (k *SkewStore) List(rehearsalID string) ([]*model.ClockSkew, error) {
	const q = `SELECT rehearsal_id,source,skew_ms,baseline FROM clock_skews WHERE rehearsal_id=?`
	rows, err := k.s.db.Query(q, rehearsalID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*model.ClockSkew
	for rows.Next() {
		var sk model.ClockSkew
		var src string
		var base int
		if err := rows.Scan(&sk.RehearsalID, &src, &sk.SkewMs, &base); err != nil {
			return nil, err
		}
		sk.Source = model.SourceKind(src)
		sk.Baseline = base != 0
		out = append(out, &sk)
	}
	return out, rows.Err()
}

// SkewOf 返回某源偏差，未知源返回 0（baseline）。
func (k *SkewStore) SkewOf(rehearsalID string, src model.SourceKind) (int64, error) {
	const q = `SELECT skew_ms FROM clock_skews WHERE rehearsal_id=? AND source=?`
	var v int64
	err := k.s.db.QueryRow(q, rehearsalID, string(src)).Scan(&v)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return v, nil
}
