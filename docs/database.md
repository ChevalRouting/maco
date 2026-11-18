# State database and job queue

Run one `maco serve` per data directory. The server and CLI commands share the
SQLite database `<data-dir>/web.db`, which holds:

- the VM state cache,
- users, API keys and SSH keys,
- backup schedules,
- metrics,
- the job queue and job history.

There is no other service to run.

## SQLite

`web.db` uses `modernc.org/sqlite` (pure Go) in WAL mode, with
`busy_timeout=10000`, `synchronous=NORMAL` and `_txlock=immediate`. Each process
shares one connection pool.

- Writers take the single SQLite write lock, across processes too, and wait up
  to 10 seconds for it before failing.
- Migrations (goose) run once per process on first use. A file lock next to the
  database stops two processes from migrating a new database at the same time.
- Keep the data directory on local storage. SQLite over a network filesystem
  is not reliable.

To back up the database while maco runs, copy the whole data directory with
maco stopped, or use SQLite's online backup on `web.db`. Either way the backup
includes job history and queued work.

## Job queue

Jobs use [goqite](https://github.com/maragudk/goqite), a queue stored in the
`goqite` table.

- `maco serve` runs one worker that executes one job at a time.
- Each job is delivered at most once. A job interrupted by a crash or restart
  is not replayed: a check every five minutes marks it failed. Inspect the
  resource and submit the operation again.
- CLI commands do not use the queue; per-VM lifecycle locks and network locks
  keep them safe next to the server.
- Preview captures and scheduled backups take an expiring lease in
  `job_claims`, so each is queued at most once per interval.

## Job history and events

Jobs and their logs live in `jobs` and `job_logs`.

- History keeps the latest 2000 jobs. Only finished jobs are trimmed, so a
  pending or running job is never dropped.
- Preview capture jobs are private: hidden from history and streams, and
  deleted after 48 hours.
- Every job change also writes a row to `job_events` in the same transaction.
  Job streams poll that table every 250 ms. Events are kept for ten minutes.
