package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// Store 封装 SQLite 连接与建表迁移。
type Store struct {
	db *sql.DB
}

// Open 打开（或创建）SQLite 数据库并建表。
func Open(path string) (*Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(1)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", path, err)
	}
	db.SetMaxOpenConns(1) // SQLite 单写者，避免跨连接写冲突
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

// DB 返回底层 *sql.DB（仅限业务包内聚合查询使用）。
func (s *Store) DB() *sql.DB { return s.db }

// Close 关闭连接。
func (s *Store) Close() error { return s.db.Close() }

func (s *Store) migrate() error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS rehearsals (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			show TEXT NOT NULL,
			state TEXT NOT NULL,
			created_at TEXT NOT NULL,
			imported_at TEXT NOT NULL,
			frozen_at TEXT NOT NULL DEFAULT ''
		);`,
		`CREATE TABLE IF NOT EXISTS stage_events (
			id TEXT PRIMARY KEY,
			rehearsal_id TEXT NOT NULL REFERENCES rehearsals(id),
			source TEXT NOT NULL,
			seq INTEGER NOT NULL DEFAULT 0,
			actor TEXT NOT NULL,
			role TEXT NOT NULL,
			label TEXT NOT NULL DEFAULT '',
			raw_timestamp INTEGER NOT NULL,
			corrected_at INTEGER NOT NULL DEFAULT 0,
			device TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_event_rehearsal_seq ON stage_events(rehearsal_id, seq);`,
		`CREATE TABLE IF NOT EXISTS clock_skews (
			rehearsal_id TEXT NOT NULL REFERENCES rehearsals(id),
			source TEXT NOT NULL,
			skew_ms INTEGER NOT NULL,
			baseline INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (rehearsal_id, source)
		);`,
		`CREATE TABLE IF NOT EXISTS constraints (
			id TEXT PRIMARY KEY,
			rehearsal_id TEXT NOT NULL REFERENCES rehearsals(id),
			name TEXT NOT NULL,
			actor TEXT NOT NULL,
			state TEXT NOT NULL,
			min_gap_ms INTEGER NOT NULL,
			note TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS conflicts (
			id TEXT PRIMARY KEY,
			rehearsal_id TEXT NOT NULL REFERENCES rehearsals(id),
			constraint_id TEXT NOT NULL REFERENCES constraints(id),
			actor TEXT NOT NULL,
			cue_event_id TEXT NOT NULL,
			mech_event_id TEXT NOT NULL,
			overlap_ms INTEGER NOT NULL,
			at_ms INTEGER NOT NULL,
			resolved INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS waivers (
			id TEXT PRIMARY KEY,
			conflict_id TEXT NOT NULL REFERENCES conflicts(id),
			reason TEXT NOT NULL,
			evidence TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS cue_packages (
			id TEXT PRIMARY KEY,
			rehearsal_id TEXT NOT NULL REFERENCES rehearsals(id),
			version INTEGER NOT NULL,
			state TEXT NOT NULL,
			snapshot_digest TEXT NOT NULL DEFAULT '',
			released_at TEXT NOT NULL DEFAULT '',
			superseded_by TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_events_rehearsal ON stage_events(rehearsal_id);
		CREATE INDEX IF NOT EXISTS idx_conflicts_rehearsal ON conflicts(rehearsal_id);
		CREATE INDEX IF NOT EXISTS idx_packages_rehearsal ON cue_packages(rehearsal_id);`,
	}
	for _, st := range stmts {
		if _, err := s.db.Exec(st); err != nil {
			return fmt.Errorf("migrate exec: %w", err)
		}
	}
	return nil
}

// nowISO 返回当前时间 ISO 字符串。
func nowISO() string { return time.Now().UTC().Format(time.RFC3339) }
