package engine

import (
	"context"
	"runtime"
	"strings"
	"time"

	"github.com/m-vinc/maco/pkg/db"
	"github.com/m-vinc/maco/pkg/metrics"
	"github.com/m-vinc/maco/pkg/vm"
	"github.com/rs/zerolog/log"
)

type metricPrevious struct {
	counters metrics.Counters
	at       time.Time
}

type metricCollector struct {
	engine   *Engine
	previous map[string]metricPrevious
}

func (e *Engine) CollectMetrics(ctx context.Context) {
	collector := metricCollector{engine: e, previous: make(map[string]metricPrevious)}
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		if err := collector.sample(ctx); err != nil {
			log.Warn().Err(err).Msg("collect utilization metrics")
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (c *metricCollector) point(key string, now time.Time, current metrics.Counters, cpus int, process bool) metrics.Point {
	point := metrics.Point{TS: now.Unix(), MemoryBytes: current.Memory, MemoryTotal: current.Capacity, RXBytes: current.RX, TXBytes: current.TX}
	if previous, ok := c.previous[key]; ok {
		seconds := now.Sub(previous.at).Seconds()
		point.CPUPercent = metrics.CPU(current, previous.counters, seconds, cpus, process)
		point.RXBytesPS = metrics.Rate(current.RX, previous.counters.RX, seconds)
		point.TXBytesPS = metrics.Rate(current.TX, previous.counters.TX, seconds)
	}

	c.previous[key] = metricPrevious{counters: current, at: now}
	return point
}

func (c *metricCollector) sample(ctx context.Context) error {
	host, err := metrics.ReadHost()
	if err != nil {
		return err
	}

	now := time.Now()
	samples := []db.MetricSample{{Scope: "host", Target: "host", Point: c.point("host", now, host, 1, false)}}
	networks, networkErr := metrics.ReadNetwork()
	if networkErr != nil {
		log.Warn().Err(networkErr).Msg("collect host network counters")
	}

	for name, counters := range networks {
		samples = append(samples, db.MetricSample{Scope: "interface", Target: name, Point: c.point("interface:"+name, now, counters, 1, false)})
	}

	manifests, err := c.engine.vms.List()
	if err != nil {
		return err
	}

	complete := true
	for _, manifest := range manifests {
		key := "vm:" + manifest.ID
		status := c.engine.driver.Status(manifest.ID)
		if status.Phase != vm.PhaseRunning {
			delete(c.previous, key)
			zero := 0.0
			samples = append(samples, db.MetricSample{Scope: "vm", Target: manifest.ID, Point: metrics.Point{TS: now.Unix(), CPUPercent: &zero, MemoryTotal: uint64(manifest.MemoryMiB) * 1024 * 1024}})
			continue
		}

		counters, err := metrics.ReadProcess(status.PID)
		if err != nil {
			complete = false
			delete(c.previous, key)
			continue
		}

		counters.Capacity = uint64(manifest.MemoryMiB) * 1024 * 1024
		samples = append(samples, db.MetricSample{Scope: "vm", Target: manifest.ID, Point: c.point(key, now, counters, runtime.NumCPU(), true)})
	}

	if complete {
		samples = append(samples, db.MetricSample{Scope: "vm", Target: "all", Point: aggregateVMMetrics(samples, now.Unix())})
	}

	active := make(map[string]bool, len(samples))
	for _, sample := range samples {
		key := sample.Scope + ":" + sample.Target
		if sample.Scope == "host" {
			key = "host"
		}

		active[key] = true
	}

	c.prunePrevious(active, networkErr == nil)

	database, err := c.engine.OpenDB(ctx)
	if err != nil {
		return err
	}

	return database.SaveMetrics(ctx, samples, now.Add(-7*24*time.Hour).Unix())
}

func (e *Engine) MetricsHistory(ctx context.Context, scope, target string, duration time.Duration) ([]metrics.Point, error) {
	database, err := e.OpenDB(ctx)
	if err != nil {
		return nil, err
	}

	bucket := max(5, int64(duration.Seconds())/720)
	return database.MetricsHistory(ctx, scope, target, time.Now().Add(-duration).Unix(), bucket)
}

func aggregateVMMetrics(samples []db.MetricSample, ts int64) metrics.Point {
	point := metrics.Point{TS: ts}
	totalCPU := 0.0
	baseline := true
	for _, sample := range samples {
		if sample.Scope != "vm" {
			continue
		}

		point.MemoryBytes += sample.Point.MemoryBytes
		point.MemoryTotal += sample.Point.MemoryTotal
		if sample.Point.CPUPercent == nil {
			baseline = false
			continue
		}

		totalCPU += *sample.Point.CPUPercent
	}

	if baseline {
		point.CPUPercent = &totalCPU
	}

	return point
}

func (c *metricCollector) prunePrevious(active map[string]bool, networkAvailable bool) {
	for key := range c.previous {
		if !active[key] && (networkAvailable || !strings.HasPrefix(key, "interface:")) {
			delete(c.previous, key)
		}
	}
}
