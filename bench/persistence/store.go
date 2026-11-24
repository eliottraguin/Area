package main

import "context"

// Store is the minimal interface each engine must offer for the scenarios.
type Store interface {
	Name() string
	Reset(ctx context.Context) error
	Close()

	// Hooks: true if the event is new.
	RecordEvent(ctx context.Context, areaID int64, extID string) (bool, error)
	CountEvents(ctx context.Context) (int64, error)

	// Runs: insert a run with a given identifier.
	InsertRun(ctx context.Context, id int64) error
	RunIDs(ctx context.Context) (map[int64]struct{}, error)

	// Jobs: queue stored in the database.
	EnqueueJobs(ctx context.Context, n int) error
	ClaimJob(ctx context.Context) (id int64, ok bool, err error)
	FinishJob(ctx context.Context, id int64) error
	CountJobsDone(ctx context.Context) (int64, error)
}

// idRows est satisfait par pgx.Rows et sql.Rows.
type idRows interface {
	Next() bool
	Scan(...any) error
	Err() error
}

func collectIDs(rows idRows) (map[int64]struct{}, error) {
	out := map[int64]struct{}{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = struct{}{}
	}
	return out, rows.Err()
}
