// Package store persists per-minute latency samples in SQLite.
package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// maxSeriesRows caps a single series query so a long range can't OOM the
// JSON encoder or the browser. ~50k samples is plenty for any visible chart.
const maxSeriesRows = 50000

type Sample struct {
	TargetID string
	Ts       int64 // unix seconds
	Min      *float64
	P50      *float64
	Max      *float64
	LossPct  float64 // 0..1
}

type MetricSample struct {
	TargetID string
	Metric   string
	Ts       int64
	Value    *float64
	Extra    string
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

		CREATE TABLE IF NOT EXISTS metric_samples (
			target_id TEXT NOT NULL,
			metric    TEXT NOT NULL,
			ts        INTEGER NOT NULL,
			value     REAL,
			extra     TEXT NOT NULL DEFAULT '',
			PRIMARY KEY (target_id, metric, ts)
		);
		CREATE INDEX IF NOT EXISTS idx_metric_samples_target_metric_ts ON metric_samples(target_id, metric, ts);
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

func (s *Store) InsertMetric(sm MetricSample) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO metric_samples (target_id, metric, ts, value, extra) VALUES (?,?,?,?,?)`,
		sm.TargetID, sm.Metric, sm.Ts, sm.Value, sm.Extra,
	)
	return err
}

func (s *Store) Series(targetID string, sinceTs int64) ([]Sample, error) {
	rows, err := s.db.Query(
		`SELECT target_id, ts, min_ms, p50_ms, max_ms, loss FROM samples
		 WHERE target_id=? AND ts>=?
		 ORDER BY ts DESC LIMIT ?`,
		targetID, sinceTs, maxSeriesRows,
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	reverseSamples(out)
	return out, nil
}

func (s *Store) MetricSeries(targetID, metric string, sinceTs int64) ([]MetricSample, error) {
	rows, err := s.db.Query(
		`SELECT target_id, metric, ts, value, extra FROM metric_samples
		 WHERE target_id=? AND metric=? AND ts>=?
		 ORDER BY ts DESC LIMIT ?`,
		targetID, metric, sinceTs, maxSeriesRows,
	)
	if err != nil {
		return nil, fmt.Errorf("query metric series: %w", err)
	}
	defer rows.Close()

	var out []MetricSample
	for rows.Next() {
		var sm MetricSample
		if err := rows.Scan(&sm.TargetID, &sm.Metric, &sm.Ts, &sm.Value, &sm.Extra); err != nil {
			return nil, err
		}
		out = append(out, sm)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	reverseMetricSamples(out)
	return out, nil
}

func reverseSamples(s []Sample) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func reverseMetricSamples(s []MetricSample) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
