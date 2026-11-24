package main

import "context"

// Store est l'interface minimale que chaque moteur doit offrir pour les scénarios.
type Store interface {
	Name() string
	Reset(ctx context.Context) error
	Close()

	// Hooks : true si l'événement est nouveau.
	RecordEvent(ctx context.Context, areaID int64, extID string) (bool, error)
	CountEvents(ctx context.Context) (int64, error)

	// Runs : insertion d'un run avec un identifiant imposé.
	InsertRun(ctx context.Context, id int64) error
	RunIDs(ctx context.Context) (map[int64]struct{}, error)

	// Jobs : file en base.
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
