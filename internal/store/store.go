// Package store persists per-minute latency samples in SQLite.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type Sample struct {
	TargetID string
	Ts       int64 // unix seconds
	Min      *float64
	P50      *float64
	Max      *float64
	LossPct  float64 // 0..1
}

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS samples (
			target_id TEXT NOT NULL,
			ts        INTEGER NOT NULL,
			min_ms    REAL,
			p50_ms    REAL,
			max_ms    REAL,
			loss      REAL NOT NULL,
			PRIMARY KEY (target_id, ts)
		);
		CREATE INDEX IF NOT EXISTS idx_samples_target_ts ON samples(target_id, ts);
	`); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Insert(sm Sample) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO samples (target_id, ts, min_ms, p50_ms, max_ms, loss) VALUES (?,?,?,?,?,?)`,
		sm.TargetID, sm.Ts, sm.Min, sm.P50, sm.Max, sm.LossPct,
	)
	return err
}

func (s *Store) Series(targetID string, sinceTs int64) ([]Sample, error) {
	rows, err := s.db.Query(
		`SELECT target_id, ts, min_ms, p50_ms, max_ms, loss FROM samples WHERE target_id=? AND ts>=? ORDER BY ts ASC`,
		targetID, sinceTs,
	)
	if err != nil {
		return nil, fmt.Errorf("query series: %w", err)
	}
	defer rows.Close()

	var out []Sample
	for rows.Next() {
		var sm Sample
		if err := rows.Scan(&sm.TargetID, &sm.Ts, &sm.Min, &sm.P50, &sm.Max, &sm.LossPct); err != nil {
			return nil, err
		}
		out = append(out, sm)
	}
	return out, rows.Err()
}
