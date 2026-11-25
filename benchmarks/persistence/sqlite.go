package main

import (
	"context"
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type sqliteStore struct {
	db   *sql.DB
	name string
}

// newSQLite opens a file database in WAL mode with synchronous=FULL.
// maxConns=1 is the tuned setup; maxConns>1 is a naive pool.
func newSQLite(path, name string, maxConns int) (Store, error) {
	dsn := fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=synchronous(FULL)", path)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxConns)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return &sqliteStore{db, name}, nil
}

func (s *sqliteStore) Name() string { return s.name }
func (s *sqliteStore) Close()       { s.db.Close() }

func (s *sqliteStore) Reset(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
DROP TABLE IF EXISTS events; DROP TABLE IF EXISTS runs; DROP TABLE IF EXISTS jobs;
CREATE TABLE events (
  id INTEGER PRIMARY KEY AUTOINCREMENT, area_id INTEGER NOT NULL, external_id TEXT NOT NULL,
  payload TEXT NOT NULL, UNIQUE (area_id, external_id));
CREATE TABLE runs (
  id INTEGER PRIMARY KEY, area_id INTEGER NOT NULL, payload TEXT NOT NULL,
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
CREATE TABLE jobs (
  id INTEGER PRIMARY KEY AUTOINCREMENT, status TEXT NOT NULL DEFAULT 'pending',
  attempts INTEGER NOT NULL DEFAULT 0);
CREATE INDEX jobs_ready ON jobs (id) WHERE status = 'pending';`)
	return err
}

func (s *sqliteStore) RecordEvent(ctx context.Context, areaID int64, extID string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `
INSERT INTO events (area_id, external_id, payload) VALUES (?, ?, ?)
ON CONFLICT (area_id, external_id) DO NOTHING`, areaID, extID, `{"from":"bench"}`)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n == 1, nil
}

func (s *sqliteStore) count(ctx context.Context, q string) (n int64, err error) {
	err = s.db.QueryRowContext(ctx, q).Scan(&n)
	return
}

func (s *sqliteStore) CountEvents(ctx context.Context) (int64, error) {
	return s.count(ctx, `SELECT count(*) FROM events`)
}

func (s *sqliteStore) InsertRun(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO runs (id, area_id, payload) VALUES (?, 1, ?)`,
		id, fmt.Sprintf(`{"run":%d,"nodes":[{"n":"a","ok":true},{"n":"b","ok":true}]}`, id))
	return err
}

func (s *sqliteStore) RunIDs(ctx context.Context) (map[int64]struct{}, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM runs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectIDs(rows)
}

func (s *sqliteStore) EnqueueJobs(ctx context.Context, n int) error {
	_, err := s.db.ExecContext(ctx, `
WITH RECURSIVE seq(i) AS (SELECT 1 UNION ALL SELECT i + 1 FROM seq WHERE i < ?)
INSERT INTO jobs (status) SELECT 'pending' FROM seq`, n)
	return err
}

// There is no SKIP LOCKED: a single UPDATE … RETURNING statement is atomic under the write lock.
func (s *sqliteStore) ClaimJob(ctx context.Context) (int64, bool, error) {
	var id int64
	err := s.db.QueryRowContext(ctx, `
UPDATE jobs SET status = 'running', attempts = attempts + 1
WHERE id = (SELECT id FROM jobs WHERE status = 'pending' ORDER BY id LIMIT 1)
RETURNING id`).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

func (s *sqliteStore) FinishJob(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `UPDATE jobs SET status = 'done' WHERE id = ?`, id)
	return err
}

func (s *sqliteStore) CountJobsDone(ctx context.Context) (int64, error) {
	return s.count(ctx, `SELECT count(*) FROM jobs WHERE status = 'done'`)
}
