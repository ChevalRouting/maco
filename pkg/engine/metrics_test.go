//go:build cgo

package engine

import (
	"context"
	"testing"
	"time"

	"github.com/m-vinc/maco/pkg/db"
	"github.com/m-vinc/maco/pkg/metrics"
)

func TestCollectorPersistsNativeMetrics(t *testing.T) {
	e := testEngine(t)
	collector := metricCollector{engine: e, previous: make(map[string]metricPrevious)}
	ctx := context.Background()
	if err := collector.sample(ctx); err != nil {
		t.Fatal(err)
	}

	if err := collector.sample(ctx); err != nil {
		t.Fatal(err)
	}

	points, err := e.MetricsHistory(ctx, "host", "host", time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	if len(points) == 0 || points[0].MemoryBytes == 0 || points[0].MemoryTotal == 0 {
		t.Fatalf("host history = %+v", points)
	}

	point := collector.point("restart", time.Now(), metrics.Counters{Busy: 1e9, Identity: 2}, 2, true)
	if point.CPUPercent != nil {
		t.Fatal("initial CPU baseline must be null")
	}

	point = collector.point("restart", time.Now(), metrics.Counters{Busy: 2e9, Identity: 3}, 2, true)
	if point.CPUPercent != nil {
		t.Fatal("changed process identity must reset CPU baseline")
	}
}

func TestAggregateVMMetrics(t *testing.T) {
	cpu1, cpu2 := 25.0, 75.0
	samples := []db.MetricSample{
		{Scope: "host", Target: "host", Point: metrics.Point{MemoryBytes: 999}},
		{Scope: "vm", Target: "one", Point: metrics.Point{CPUPercent: &cpu1, MemoryBytes: 100, MemoryTotal: 500}},
		{Scope: "vm", Target: "two", Point: metrics.Point{CPUPercent: &cpu2, MemoryBytes: 200, MemoryTotal: 1000}},
	}
	point := aggregateVMMetrics(samples, 100)
	if point.CPUPercent == nil || *point.CPUPercent != 100 || point.MemoryBytes != 300 || point.MemoryTotal != 1500 {
		t.Fatalf("global VM statistics = %+v", point)
	}

	samples[1].Point.CPUPercent = nil
	if aggregateVMMetrics(samples, 100).CPUPercent != nil {
		t.Fatal("global CPU must wait for every VM baseline")
	}

	empty := aggregateVMMetrics(nil, 100)
	if empty.CPUPercent == nil || *empty.CPUPercent != 0 || empty.MemoryBytes != 0 {
		t.Fatalf("empty VM statistics = %+v", empty)
	}
}

func TestTransientNetworkFailureKeepsBaseline(t *testing.T) {
	collector := metricCollector{previous: map[string]metricPrevious{
		"interface:en0": {counters: metrics.Counters{RX: 100, TX: 200}, at: time.Unix(100, 0)},
		"vm:removed":    {},
	}}
	collector.prunePrevious(nil, false)
	if _, exists := collector.previous["vm:removed"]; exists {
		t.Fatal("removed VM baseline should be discarded")
	}

	point := collector.point("interface:en0", time.Unix(110, 0), metrics.Counters{RX: 200, TX: 400}, 1, false)
	if point.RXBytesPS == nil || *point.RXBytesPS != 10 || point.TXBytesPS == nil || *point.TXBytesPS != 20 {
		t.Fatalf("network rates after transient failure = %+v", point)
	}

	collector.prunePrevious(nil, true)
	if _, exists := collector.previous["interface:en0"]; exists {
		t.Fatal("successfully observed interface removal should discard baseline")
	}
}
