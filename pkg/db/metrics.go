package db

import (
	"context"

	"github.com/m-vinc/maco/pkg/metrics"
)

type MetricSample struct {
	Scope  string
	Target string
	Point  metrics.Point
}

func (db *DB) SaveMetrics(ctx context.Context, samples []MetricSample, cutoff int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer func() { _ = tx.Rollback() }()
	for _, sample := range samples {
		p := sample.Point
		_, err := tx.ExecContext(ctx, `INSERT OR REPLACE INTO metric_samples (scope, target, ts, cpu_percent, memory_bytes, memory_total, rx_bytes, tx_bytes, rx_bytes_ps, tx_bytes_ps, cpu_basis, counter_source) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'host-time', 'mib')`, sample.Scope, sample.Target, p.TS, p.CPUPercent, p.MemoryBytes, p.MemoryTotal, p.RXBytes, p.TXBytes, p.RXBytesPS, p.TXBytesPS)
		if err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM metric_samples WHERE ts < ?`, cutoff); err != nil {
		return err
	}

	return tx.Commit()
}

func (db *DB) MetricsHistory(ctx context.Context, scope, target string, since int64, bucket int64) ([]metrics.Point, error) {
	rows, err := db.QueryContext(ctx, `
WITH history AS (
    SELECT *, LAG(ts) OVER (ORDER BY ts) AS previous_ts,
        LAG(rx_bytes) OVER (ORDER BY ts) AS previous_rx,
        LAG(tx_bytes) OVER (ORDER BY ts) AS previous_tx
    FROM metric_samples WHERE scope = ? AND target = ? AND ts >= ?
), samples AS (
    SELECT *, (ts / ?) * ? AS bucket,
        ROW_NUMBER() OVER (PARTITION BY ts / ? ORDER BY ts DESC) AS position,
        COALESCE(rx_bytes_ps, CASE WHEN counter_source = 'route'
            AND previous_rx BETWEEN 3221225472 AND 4294967295 AND rx_bytes < 1073741824
            AND ts - previous_ts BETWEEN 1 AND 30
            THEN (4294967296.0 - previous_rx + rx_bytes) / (ts - previous_ts) END) AS rx_rate,
        COALESCE(tx_bytes_ps, CASE WHEN counter_source = 'route'
            AND previous_tx BETWEEN 3221225472 AND 4294967295 AND tx_bytes < 1073741824
            AND ts - previous_ts BETWEEN 1 AND 30
            THEN (4294967296.0 - previous_tx + tx_bytes) / (ts - previous_ts) END) AS tx_rate
    FROM history WHERE ts >= ?
)
SELECT bucket, AVG(CASE WHEN scope = 'vm' AND cpu_basis != 'host-time' THEN NULL ELSE cpu_percent END),
    CAST(AVG(memory_bytes) AS INTEGER), CAST(AVG(memory_total) AS INTEGER),
    MAX(CASE WHEN position = 1 THEN rx_bytes END), MAX(CASE WHEN position = 1 THEN tx_bytes END),
    AVG(rx_rate), AVG(tx_rate)
FROM samples GROUP BY bucket ORDER BY bucket`, scope, target, since-30, bucket, bucket, bucket, since)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()
	points := make([]metrics.Point, 0)
	for rows.Next() {
		var p metrics.Point
		if err := rows.Scan(&p.TS, &p.CPUPercent, &p.MemoryBytes, &p.MemoryTotal, &p.RXBytes, &p.TXBytes, &p.RXBytesPS, &p.TXBytesPS); err != nil {
			return nil, err
		}

		points = append(points, p)
	}

	return points, rows.Err()
}

func (db *DB) MetricInterfaces(ctx context.Context) ([]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT target FROM metric_samples WHERE scope = 'interface' ORDER BY target`)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()
	names := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}

		names = append(names, name)
	}

	return names, rows.Err()
}
