package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"maragu.dev/goqite"
	"maragu.dev/goqite/jobs"
)

const jobColumns = `id, action, target, state, created_at, started_at, finished_at, error, result`

type rowScanner interface {
	Scan(dest ...any) error
}

func millis(value *time.Time) any {
	if value == nil {
		return nil
	}

	return value.UnixMilli()
}

func fromMillis(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}

	moment := time.UnixMilli(value.Int64).UTC()
	return &moment
}

func scanJob(row rowScanner) (*Job, error) {
	var (
		j        Job
		state    string
		created  int64
		started  sql.NullInt64
		finished sql.NullInt64
	)

	if err := row.Scan(&j.ID, &j.Action, &j.Target, &state, &created, &started, &finished, &j.Error, &j.Result); err != nil {
		return nil, err
	}

	j.State, j.CreatedAt = State(state), time.UnixMilli(created).UTC()
	j.StartedAt, j.FinishedAt = fromMillis(started), fromMillis(finished)
	j.Label, j.Logs = Label(j.Action), []string{}
	return &j, nil
}

func publish(ctx context.Context, tx *sql.Tx, j *Job) error {
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO job_events (payload, created_at) VALUES (?, ?)`, string(data), time.Now().UnixMilli())
	return err
}

func (s *Service) insert(ctx context.Context, j *Job, body []byte) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()
	message, err := jobs.CreateTx(ctx, tx, s.queue, actionJob, goqite.Message{Body: body})
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, `INSERT INTO jobs (id, action, target, state, private, message_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`, j.ID, j.Action, j.Target, string(j.State), j.Private(), string(message), j.CreatedAt.UnixMilli())
	if err != nil {
		return err
	}

	if err := publish(ctx, tx, j); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Service) save(ctx context.Context, j *Job) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `UPDATE jobs SET state = ?, started_at = ?, finished_at = ?, error = ?, result = ? WHERE id = ?`, string(j.State), millis(j.StartedAt), millis(j.FinishedAt), j.Error, j.Result, j.ID)
	if err != nil {
		return err
	}

	if err := publish(ctx, tx, j); err != nil {
		return err
	}

	return tx.Commit()
}

func (s *Service) appendLog(ctx context.Context, id, message string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO job_logs (job_id, message) VALUES (?, ?)`, id, time.Now().UTC().Format(time.RFC3339)+" "+message)
	return err
}

func (s *Service) Get(ctx context.Context, id string) (*Job, error) {
	j, err := scanJob(s.db.QueryRowContext(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id = ?`, id))
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `SELECT message FROM job_logs WHERE job_id = ? ORDER BY id`, id)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var message string
		if err := rows.Scan(&message); err != nil {
			return nil, err
		}

		j.Logs = append(j.Logs, message)
	}

	return j, rows.Err()
}

func (s *Service) queryJobs(ctx context.Context, query string, args ...any) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()
	list := make([]Job, 0)
	for rows.Next() {
		j, err := scanJob(rows)
		if err != nil {
			return nil, err
		}

		list = append(list, *j)
	}

	return list, rows.Err()
}

func (s *Service) List(ctx context.Context) ([]Job, error) {
	return s.queryJobs(ctx, `SELECT `+jobColumns+` FROM jobs WHERE id IN (SELECT id FROM jobs WHERE private = 0 ORDER BY created_at DESC LIMIT 200) OR state IN ('pending', 'running') ORDER BY created_at DESC`)
}

func (s *Service) PublicList(ctx context.Context) ([]Job, error) {
	return s.queryJobs(ctx, `SELECT `+jobColumns+` FROM jobs WHERE private = 0 AND (id IN (SELECT id FROM jobs WHERE private = 0 ORDER BY created_at DESC LIMIT 200) OR state IN ('pending', 'running')) ORDER BY created_at DESC`)
}

func (s *Service) publicJobs(ctx context.Context) ([]Job, error) {
	return s.queryJobs(ctx, `SELECT `+jobColumns+` FROM jobs WHERE private = 0 ORDER BY created_at DESC`)
}

func (s *Service) trimHistory(ctx context.Context) {
	_, _ = s.db.ExecContext(ctx, `DELETE FROM jobs WHERE private = 0 AND state IN ('succeeded', 'failed') AND id NOT IN (SELECT id FROM jobs WHERE private = 0 ORDER BY created_at DESC LIMIT ?)`, historyLimit)
}

func (s *Service) purge(ctx context.Context) {
	now := time.Now()
	_, _ = s.db.ExecContext(ctx, `DELETE FROM jobs WHERE private = 1 AND created_at < ?`, now.Add(-privateJobTTL).UnixMilli())
	_, _ = s.db.ExecContext(ctx, `DELETE FROM job_events WHERE created_at < ?`, now.Add(-eventRetention).UnixMilli())
	_, _ = s.db.ExecContext(ctx, `DELETE FROM job_claims WHERE expires_at < ?`, now.UnixMilli())
	_, _ = s.db.ExecContext(ctx, `DELETE FROM goqite WHERE queue = 'maco' AND received > 0 AND timeout < strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-1 hour')`)
}

func (s *Service) reconcileOrphans(ctx context.Context) {
	orphans, err := s.queryJobs(ctx, `SELECT `+jobColumns+` FROM jobs WHERE state IN ('pending', 'running') AND created_at < ? AND NOT EXISTS (SELECT 1 FROM goqite WHERE goqite.id = jobs.message_id AND (goqite.received = 0 OR goqite.timeout > strftime('%Y-%m-%dT%H:%M:%fZ', 'now')))`, time.Now().Add(-time.Minute).UnixMilli())
	if err != nil {
		return
	}

	for index := range orphans {
		job := &orphans[index]
		job.Error = "job was interrupted before it finished"
		if job.StartedAt == nil {
			job.Error = "job was interrupted before it started"
		}

		job.State = Failed
		now := time.Now().UTC()
		job.FinishedAt = &now
		_ = s.appendLog(ctx, job.ID, "Failed: "+job.Error)
		_ = s.save(ctx, job)
	}
}

func (s *Service) claim(ctx context.Context, key, value string, ttl time.Duration) (bool, error) {
	now := time.Now()
	result, err := s.db.ExecContext(ctx, `INSERT INTO job_claims (key, value, expires_at) VALUES (?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at WHERE job_claims.expires_at < ?`, key, value, now.Add(ttl).UnixMilli(), now.UnixMilli())
	if err != nil {
		return false, err
	}

	changed, err := result.RowsAffected()
	return changed == 1, err
}

func (s *Service) setClaim(ctx context.Context, key, value string, ttl time.Duration) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO job_claims (key, value, expires_at) VALUES (?, ?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value, expires_at = excluded.expires_at`, key, value, time.Now().Add(ttl).UnixMilli())
	return err
}

func (s *Service) claimValue(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, `SELECT value FROM job_claims WHERE key = ? AND expires_at >= ?`, key, time.Now().UnixMilli()).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}

	return value, err
}
