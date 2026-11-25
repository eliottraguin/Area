package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pgStore struct{ pool *pgxpool.Pool }

func newPG(ctx context.Context, dsn string) (Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 20
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := p.Ping(ctx); err != nil {
		return nil, err
	}
	return &pgStore{p}, nil
}

func (s *pgStore) Name() string { return "postgres" }
func (s *pgStore) Close()       { s.pool.Close() }

func (s *pgStore) Reset(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
DROP TABLE IF EXISTS events, runs, jobs;
CREATE TABLE events (
  id BIGSERIAL PRIMARY KEY, area_id BIGINT NOT NULL, external_id TEXT NOT NULL,
  payload JSONB NOT NULL, UNIQUE (area_id, external_id));
CREATE TABLE runs (
  id BIGINT PRIMARY KEY, area_id BIGINT NOT NULL, payload JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE TABLE jobs (
  id BIGSERIAL PRIMARY KEY, status TEXT NOT NULL DEFAULT 'pending',
  attempts INT NOT NULL DEFAULT 0, run_at TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE INDEX jobs_ready ON jobs (run_at) WHERE status = 'pending';`)
	return err
}

func (s *pgStore) RecordEvent(ctx context.Context, areaID int64, extID string) (bool, error) {
	tag, err := s.pool.Exec(ctx, `
INSERT INTO events (area_id, external_id, payload) VALUES ($1, $2, $3::jsonb)
ON CONFLICT (area_id, external_id) DO NOTHING`, areaID, extID, `{"from":"bench"}`)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() == 1, nil
}

func (s *pgStore) count(ctx context.Context, q string) (n int64, err error) {
	err = s.pool.QueryRow(ctx, q).Scan(&n)
	return
}

func (s *pgStore) CountEvents(ctx context.Context) (int64, error) {
	return s.count(ctx, `SELECT count(*) FROM events`)
}

func (s *pgStore) InsertRun(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO runs (id, area_id, payload) VALUES ($1, 1, $2::jsonb)`,
		id, fmt.Sprintf(`{"run":%d,"nodes":[{"n":"a","ok":true},{"n":"b","ok":true}]}`, id))
	return err
}

func (s *pgStore) RunIDs(ctx context.Context) (map[int64]struct{}, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM runs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return collectIDs(rows)
}

func (s *pgStore) EnqueueJobs(ctx context.Context, n int) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO jobs (status) SELECT 'pending' FROM generate_series(1, $1)`, n)
	return err
}

func (s *pgStore) ClaimJob(ctx context.Context) (int64, bool, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
UPDATE jobs SET status = 'running', attempts = attempts + 1
WHERE id = (
  SELECT id FROM jobs WHERE status = 'pending' AND run_at <= now()
  ORDER BY run_at, id FOR UPDATE SKIP LOCKED LIMIT 1)
RETURNING id`).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, false, nil
		}
		return 0, false, err
	}
	return id, true, nil
}

func (s *pgStore) FinishJob(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE jobs SET status = 'done' WHERE id = $1`, id)
	return err
}

func (s *pgStore) CountJobsDone(ctx context.Context) (int64, error) {
	return s.count(ctx, `SELECT count(*) FROM jobs WHERE status = 'done'`)
}
