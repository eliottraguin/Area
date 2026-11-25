# ADR-0002: Use PostgreSQL for persistence

- **Status:** Proposed
- **Date:** 2026-11-24
- **Deciders:** AREA team (four people)

## Context

AREA stores and handles:

| Data | Nature | Constraints |
|---|---|---|
| Users, roles | relational | unique email, account deletion (GDPR) |
| Connected services + OAuth tokens | relational + secret | encryption at rest, rotation |
| Workflows (AREA) and nodes | graph / document | structure varies with the node type, versioned |
| Events detected by hooks | frequent writes | deduplication: the same event must never be processed twice |
| Workflow runs (history) | append-only | inputs/outputs of each node, duration, status, metrics |
| Pending work / retry / dead letter | queue | no loss, replayable, side effects not duplicated |

Cross-cutting requirements from the subject: autonomous hooks (no client open), timestamped backups outside the database container, a health endpoint that reports the database, encryption of what a dump reveals, an inventory of personal data.

## Decision drivers

| # | Criterion | Weight | Why |
|---|---|---|---|
| C1 | Integrity / transactions (ACID) | 3 | a run must never be half recorded |
| C2 | Write concurrency | 3 | hooks, workers and the API write in parallel |
| C3 | Deduplication / idempotence | 3 | core requirement of hooks and retry |
| C4 | Modeling (relational + semi-structured) | 2 | workflows with heterogeneous nodes |
| C5 | Security (encryption, permissions, attack surface) | 3 | OAuth tokens, database dump |
| C6 | Docker operations (backup, health, container separation) | 2 | additional ENG features |
| C7 | Go ecosystem support | 2 | mature driver, easy testing |
| C8 | Job queue without a broker | 2 | retry + dead letter without an extra component |
| C9 | Team learning curve | 1 | one week for the Plan milestone |

## Options considered

PostgreSQL, SQLite and MongoDB.

### Overview

| | PostgreSQL | SQLite | MongoDB |
|---|---|---|---|
| Model | relational (+ JSONB) | embedded relational | documents (BSON) |
| Architecture | server | file, in-process | server |
| Persistence | WAL, durable | journal (WAL or rollback), durable | journal + checkpoints, durable |
| Transactions | full, multi-table | full | multi-document only on a replica set |
| Concurrent writes | MVCC, many writers | one writer at a time | document-level locking |
| Uniqueness constraint | yes | yes | unique index |
| `INSERT … ON CONFLICT` | yes | yes (≥ 3.24) | upsert / `E11000` |
| Claiming jobs without duplicates | `FOR UPDATE SKIP LOCKED` | no (global lock) | atomic `findOneAndUpdate` |
| Go driver | `pgx` (very mature) | `mattn/go-sqlite3` (cgo) or `modernc.org/sqlite` (pure Go) | `mongo-go-driver` (official) |
| License | PostgreSQL (permissive) | public domain | SSPL (server) |
| Access from several containers | yes (network) | impractical (shared file) | yes (network) |

### Scoring (0 to 5, weighted)

| Criterion | Weight | PostgreSQL | SQLite | MongoDB |
|---|---|---|---|---|
| C1 ACID | 3 | 5 | 5 | 3 |
| C2 Write concurrency | 3 | 5 | 2 | 4 |
| C3 Idempotence | 3 | 5 | 4 | 4 |
| C4 Modeling | 2 | 4 | 3 | 5 |
| C5 Security | 3 | 4 | 3 | 3 |
| C6 Docker operations | 2 | 5 | 2 | 4 |
| C7 Go ecosystem | 2 | 5 | 4 | 4 |
| C8 Queue without a broker | 2 | 5 | 2 | 3 |
| C9 Learning curve | 1 | 3 | 5 | 3 |
| Weighted total (/105) | | 98 (≈ 93%) | 69 (≈ 66%) | 77 (≈ 73%) |

### Analysis by criterion

#### Hook idempotence (C3)

A hook detects an event (new email, new commit…). It must record the event exactly once, even if polling sees it again.

With PostgreSQL:

```sql
CREATE TABLE events (
    id          BIGSERIAL PRIMARY KEY,
    area_id     BIGINT      NOT NULL REFERENCES areas(id) ON DELETE CASCADE,
    external_id TEXT        NOT NULL,          -- identifier on the service side (mail id, commit sha…)
    payload     JSONB       NOT NULL,
    seen_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (area_id, external_id)
);
```

```go
// true if the event is new, false if it was already known
func (s *Store) RecordEvent(ctx context.Context, areaID int64, extID string, payload []byte) (bool, error) {
    tag, err := s.pool.Exec(ctx, `
        INSERT INTO events (area_id, external_id, payload)
        VALUES ($1, $2, $3)
        ON CONFLICT (area_id, external_id) DO NOTHING`, areaID, extID, payload)
    if err != nil {
        return false, err
    }
    return tag.RowsAffected() == 1, nil
}
```

The database enforces deduplication through a uniqueness constraint, so it stays correct even with several hook instances.

SQLite accepts the same query (`ON CONFLICT DO NOTHING`). Its weak point is concurrency (see "Write concurrency").

With MongoDB:

```go
_, err := coll.InsertOne(ctx, bson.M{"area_id": areaID, "external_id": extID, "payload": payload})
if mongo.IsDuplicateKeyError(err) {
    return false, nil // already seen
}
```

The unique index must exist beforehand: `db.events.createIndex({area_id: 1, external_id: 1}, {unique: true})`.

#### Write concurrency (C2)

In AREA, the API (clients), the hooks and the workers write at the same time.

SQLite allows only one writer. WAL mode lets readers continue, but writes are serialized:

```go
db, _ := sql.Open("sqlite", "file:area.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
db.SetMaxOpenConns(1) // otherwise SQLITE_BUSY under load
```

The pool must therefore be capped and every write routed through a single queue. This works at small scale, but only one process can safely hold the file, which constrains the architecture.

PostgreSQL (MVCC) handles parallel writes without special configuration.

MongoDB locks at the document level and also handles parallel writes (7,777 ops/s with 16 goroutines, see "Practical evidence").

#### Job queue, retry and dead letter without a broker (C8)

The subject requires that retry does not need a broker (ENG-RETRY). With PostgreSQL, the table itself serves as the queue:

```sql
CREATE TABLE jobs (
    id            BIGSERIAL PRIMARY KEY,
    run_id        BIGINT      NOT NULL,
    node_id       TEXT        NOT NULL,
    status        TEXT        NOT NULL DEFAULT 'pending',  -- pending | running | done | dead
    attempts      INT         NOT NULL DEFAULT 0,
    max_attempts  INT         NOT NULL DEFAULT 5,
    run_at        TIMESTAMPTZ NOT NULL DEFAULT now(),       -- backoff: pushed back on each failure
    idempotency_key TEXT      NOT NULL UNIQUE,              -- protects the side effect
    last_error    TEXT
);
CREATE INDEX jobs_ready ON jobs (run_at) WHERE status = 'pending';
```

```sql
-- Atomic claim by several workers, without double processing
UPDATE jobs SET status = 'running', attempts = attempts + 1
WHERE id = (
    SELECT id FROM jobs
    WHERE status = 'pending' AND run_at <= now()
    ORDER BY run_at
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;
```

```sql
-- Failure: exponential backoff, or dead letter once attempts are exhausted
UPDATE jobs SET
    status  = CASE WHEN attempts >= max_attempts THEN 'dead' ELSE 'pending' END,
    run_at  = now() + (interval '2 seconds' * power(2, attempts)),
    last_error = $2
WHERE id = $1;
```

`dead` jobs stay in the database, so they can be inspected and replayed (`UPDATE jobs SET status='pending', attempts=0 …`). The link between retry, persistence and idempotence that the subject asks about is direct here: the idempotency key is a uniqueness constraint in the same transaction as the job state.

SQLite has no `SKIP LOCKED`. With a single writer, claiming is safe inside one process, but not across several.

MongoDB claims jobs with `findOneAndUpdate`, which is atomic, but keeping the job and the side effect atomic together requires a transaction, hence a replica set.

#### Workflow modeling (C4)

Nodes have heterogeneous parameters. PostgreSQL combines relational columns and JSONB:

```sql
CREATE TABLE nodes (
    id        BIGSERIAL PRIMARY KEY,
    area_id   BIGINT NOT NULL REFERENCES areas(id) ON DELETE CASCADE,
    kind      TEXT   NOT NULL,        -- 'action' | 'reaction' | 'filter' | …
    service   TEXT   NOT NULL,
    params    JSONB  NOT NULL,
    next_id   BIGINT REFERENCES nodes(id)
);
CREATE INDEX nodes_params_gin ON nodes USING GIN (params);
```

MongoDB is more natural for a nested workflow document, but most of AREA's relations (user ↔ services ↔ workflows ↔ runs) are relational, with joins and cascading deletes (deleted account = deleted data). SQLite offers JSON (`json_extract`) but without equally rich indexing.

#### Security (C5)

| Aspect | PostgreSQL | SQLite | MongoDB |
|---|---|---|---|
| Authentication / roles | roles, `GRANT`, row-level security | none (OS file permissions) | users, RBAC |
| Transport encryption | TLS | not applicable (local) | TLS |
| Encryption at rest | column level (`pgcrypto`) or volume | paid extension or volume | Enterprise only; otherwise volume |
| Parameterized queries | yes (`pgx`) | yes | yes (NoSQL injection through `$` operators if raw input is passed) |
| Attack surface | network port to isolate | file to protect | network port to isolate |

Design choice: application-level encryption of OAuth tokens (AES-256-GCM, with the key kept outside the repository and the database, supplied through an environment variable or a Docker secret). A database dump then reveals only ciphertext, whichever engine is used.

```go
func Encrypt(key, plaintext []byte) ([]byte, error) {
    block, _ := aes.NewCipher(key) // 32-byte key
    gcm, _ := cipher.NewGCM(block)
    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, err
    }
    return gcm.Seal(nonce, nonce, plaintext, nil), nil // nonce || ciphertext
}
```

This choice makes C5 less discriminating between engines. The remaining difference is roles: PostgreSQL allows an application user without `DROP` rights, or with read-only access to an `audit` table.

#### Docker operations (C6)

```yaml
services:
  db:
    image: postgres:17
    environment:
      POSTGRES_PASSWORD_FILE: /run/secrets/db_password
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U area"]
      interval: 5s
  backup:
    image: postgres:17
    # timestamped pg_dump to a volume separate from the database container
    command: >
      sh -c 'while true; do
        pg_dump -h db -U area area | gzip > /backups/area-$$(date +%F_%H%M).sql.gz;
        sleep 86400; done'
    volumes:
      - backups:/backups
volumes:
  pgdata:
  backups:
```

- Health endpoint: a `pool.Ping(ctx)` call on the Go side is enough to report a stopped database.
- Backups: `pg_dump` in a dedicated container. With SQLite, a file in use by the application would have to be copied (`.backup` / `VACUUM INTO`) from another container, which means sharing the volume. MongoDB uses `mongodump`.
- Docker persistence: all three engines need a named volume, otherwise a recreated container loses its data.

### Practical evidence (POC)

The POC lives in `bench/persistence/` (Go and Docker Compose). It runs the same scenarios on the three engines, behind a common `Store` interface.

```
nix develop                        # Go, psql, sqlite3
cd bench/persistence
docker compose up -d --wait        # postgres:17 and mongo:8
go run . bench                     # scenarios 1 to 3
./crash.sh                         # scenario 4
```

Scenarios:
1. Deduplication: 5,000 distinct identifiers, each submitted twice (10,000 calls, shuffled), 16 goroutines. Expected: exactly 5,000 rows.
2. Concurrent writes: 1, 4 and 16 goroutines insert runs for 5 s. Throughput and errors are measured.
3. Job claiming: 10,000 jobs, 4 workers that claim then finish them. Expected: no job processed twice.
4. Crash: 8 goroutines write continuously and record an identifier only once the write is acknowledged. The engine is killed (`docker kill`, SIGKILL), restarted, and every acknowledged write is then checked for existence.

Variants tested: `sqlite` (WAL, `synchronous=FULL`, pool of 1 connection), `sqlite-pool` (same settings, pool of 16 connections, so without precautions), `mongo` (`j:true`, write acknowledged after the journal flush) and `mongo-w1` (default write concern).

#### Measured results (2026-11-24, average of 5 runs)

Values in parentheses give the minimum and maximum observed when the spread is notable.

| Scenario | PostgreSQL | SQLite (pool 1) | SQLite (pool 16) | MongoDB (`j:true`) |
|---|---|---|---|---|
| Rows after deduplication (expected 5,000) | 5,000 | 5,000 | 5,000 | 5,000 |
| Errors during deduplication | 0 | 0 | 14 (`SQLITE_BUSY`) | 0 |
| Deduplication duration | 0.67 s | 11.7 s | 12.1 s | 1.5 s |
| Write throughput, 1 goroutine (ops/s) | 870 (71 to 1,238) | 858 (832 to 878) | 789 (452 to 878) | 477 (95 to 791) |
| Write throughput, 4 goroutines (ops/s) | 2,241 (1,153 to 2,521) | 870 | 753 (291 to 877) | 2,400 |
| Write throughput, 16 goroutines (ops/s) | 9,929 (9,851 to 9,964) | 866 | 709 (74 to 881) | 7,777 (7,421 to 7,983) |
| Write errors (16 goroutines) | 0 | 0 | 6 (`SQLITE_BUSY`) | 0 |
| Jobs processed twice (expected 0) | 0 | 0 | 0 | 0 |
| Jobs done in the database (expected 10,000) | 10,000 | 10,000 | 9,998 (9,996 to 10,000) | 10,000 |
| Time to process the 10,000 jobs | 18.2 s (12.9 to 23.2) | 23.9 s | 26.8 s (23.9 to 37.0) | 8.3 s |

| Crash (`kill` during writes, average of 5 runs) | Acknowledged writes | Lost |
|---|---|---|
| PostgreSQL | 19,404 | 0 in all 5 runs |
| SQLite (`synchronous=FULL`) | 3,289 | 0 in all 5 runs |
| MongoDB `j:true` | 17,005 | 0 in all 5 runs |
| MongoDB `w:1` (default) | 85,584 | 123 on average (22 to 266) |

#### Reading the results

- Concurrency: SQLite stays around 860 ops/s whatever the number of goroutines, since it has only one writer at a time. PostgreSQL goes from 870 to 9,929 ops/s between 1 and 16 goroutines (grouped fsyncs benefit from parallelism). MongoDB follows the same trend, a bit lower (477 to 7,777 ops/s).
- Naive SQLite pool: with 16 connections, `SQLITE_BUSY` errors appear despite `busy_timeout=5000`, and a few jobs stay stuck in the `running` state (9,998 of 10,000 done on average) because the write that finishes the job failed. Throughput also becomes unstable (74 ops/s in the worst case with 16 goroutines). The pool must therefore be capped at 1 connection and errors handled.
- With a single goroutine, SQLite and PostgreSQL are on par on average (858 against 870 ops/s), but PostgreSQL is very irregular (71 to 1,238 ops/s) while SQLite varies little (832 to 878). This does not change the choice, because AREA has several concurrent writers.
- Idempotence: all three engines keep 5,000 rows thanks to the uniqueness constraint. This criterion does not separate them.
- No duplicate jobs: none in any case, including SQLite without `SKIP LOCKED` (thanks to the atomic `UPDATE … RETURNING`, as long as there is a single application instance).
- MongoDB durability: with the default write concern, 123 acknowledged writes on average (22 to 266 depending on the run) were lost after a kill. `j:true` must be configured, which lowers throughput. PostgreSQL is durable without tuning.
- MongoDB is faster on jobs (8.3 s against 18.2 s): claiming is a single operation (`findOneAndUpdate`), whereas PostgreSQL waits for two fsyncs (claim, then job completion). This point favors MongoDB. It weighs little against the other criteria, but it will have to be acknowledged during the defense.

#### Limits of the protocol

- Single machine, 5 runs: the averages give orders of magnitude. Throughput with 1 and 4 goroutines varies a lot from run to run (disk fsync, machine activity) and does not separate the engines; throughput with 16 goroutines is stable.
- Local network: the containers run on `localhost`, so the network latency of server engines is underestimated compared with a real deployment.
- `docker kill` simulates a process crash, not a power cut: the OS disk cache survives. The test checks journaling, not hardware resilience.
- Minimal schema: one row per run, without the full inputs/outputs of each node.
- Versions: PostgreSQL 17, MongoDB 8, SQLite through `modernc.org/sqlite` v1.63.0 (pure Go), Go 1.27.3, `pgx` v5, `mongo-driver` v2.

## Decision

PostgreSQL, accessed through `pgx`.

Reasons specific to AREA:
1. Hooks, workers and the API write in parallel: MVCC works without workarounds (C2).
2. Idempotence and the retry queue rely on uniqueness constraints and `SKIP LOCKED` in the same transaction as the business state, with no broker (C3, C8).
3. The model is mostly relational, and JSONB covers the variability of nodes (C4).
4. A separate server fits naturally with backups, health checks and the container breakdown (C6).

### Alternatives rejected, for this project

- SQLite: rejected because of its single-writer model against concurrent hooks, workers and API, and because a file would have to be shared between containers. It is durable and ACID, and stays relevant for unit tests.
- MongoDB: rejected because the model is mostly relational, because multi-document transactions require a replica set (operational complexity for little gain), and because the SSPL license and the Enterprise-only native encryption at rest are less favorable.

## Consequences

### Accepted consequences

- One more server container to configure, secure and back up.
- A job queue stored in the database: fine for our volume, to be reassessed if throughput grows a lot (see ENG-ASYNC).
- Schema migrations to manage (plan on `golang-migrate` or `goose`).

### Conditions for review

Revisit this choice if the evaluation shows that JSONB is not enough for workflows, or if a real load exceeds what the database can handle as a job queue.

## References

- [POC code](../../bench/persistence/)
- [ADR-0001: backend language](0001-backend-language.md)
