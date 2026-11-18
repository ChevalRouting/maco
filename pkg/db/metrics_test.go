package db

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/m-vinc/maco/pkg/metrics"
)

func TestMetricsHistoryAndRetention(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "metrics.db")
	database, err := InitDB(ctx, path)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = database.Close() }()
	cpu, rate := 50.0, 200.0
	samples := []MetricSample{
		{Scope: "host", Target: "host", Point: metrics.Point{TS: 10, MemoryBytes: 100}},
		{Scope: "host", Target: "host", Point: metrics.Point{TS: 100, MemoryBytes: 200}},
		{Scope: "host", Target: "host", Point: metrics.Point{TS: 105, CPUPercent: &cpu, MemoryBytes: 400}},
		{Scope: "interface", Target: "en0", Point: metrics.Point{TS: 100, RXBytes: 1000, RXBytesPS: &rate}},
		{Scope: "interface", Target: "en0", Point: metrics.Point{TS: 105, RXBytes: 50}},
		{Scope: "vm", Target: "vm1", Point: metrics.Point{TS: 100, CPUPercent: &cpu}},
	}
	if err := database.SaveMetrics(ctx, samples, 50); err != nil {
		t.Fatal(err)
	}

	points, err := database.MetricsHistory(ctx, "host", "host", 0, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(points) != 1 || points[0].MemoryBytes != 300 || points[0].CPUPercent == nil || *points[0].CPUPercent != 50 || points[0].RXBytesPS != nil {
		t.Fatalf("unexpected history: %+v", points)
	}

	interfaces, err := database.MetricInterfaces(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(interfaces) != 1 || interfaces[0] != "en0" {
		t.Fatalf("interfaces = %v", interfaces)
	}

	if err := database.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := InitDB(ctx, path)
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = reopened.Close() }()
	persisted, err := reopened.MetricsHistory(ctx, "interface", "en0", 0, 10)
	if err != nil {
		t.Fatal(err)
	}

	if len(persisted) != 1 || persisted[0].RXBytes != 50 || persisted[0].RXBytesPS == nil || *persisted[0].RXBytesPS != 200 {
		t.Fatalf("persisted history = %+v", persisted)
	}

	if err := reopened.SaveMetrics(ctx, nil, 106); err != nil {
		t.Fatal(err)
	}

	var remaining int
	if err := reopened.QueryRowContext(ctx, "SELECT COUNT(*) FROM metric_samples").Scan(&remaining); err != nil {
		t.Fatal(err)
	}

	if remaining != 0 {
		t.Fatalf("expired samples remain across scopes: %d", remaining)
	}
}

func TestLegacyCounterRolloverHistory(t *testing.T) {
	ctx := context.Background()
	database, err := InitDB(ctx, filepath.Join(t.TempDir(), "metrics.db"))
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = database.Close() }()
	_, err = database.ExecContext(ctx, `INSERT INTO metric_samples
        (scope, target, ts, cpu_percent, memory_bytes, memory_total, rx_bytes, tx_bytes)
        VALUES ('interface','en0',100,NULL,0,0,4200000000,50),
               ('interface','en0',105,NULL,0,0,100000000,55),
               ('vm','old',100,50,200,500,0,0)`)
	if err != nil {
		t.Fatal(err)
	}

	points, err := database.MetricsHistory(ctx, "interface", "en0", 105, 5)
	if err != nil {
		t.Fatal(err)
	}

	expected := (4294967296.0 - 4200000000 + 100000000) / 5
	if len(points) != 1 || points[0].RXBytesPS == nil || *points[0].RXBytesPS != expected || points[0].TXBytesPS != nil {
		t.Fatalf("counter rollover history = %+v", points)
	}

	old, err := database.MetricsHistory(ctx, "vm", "old", 0, 5)
	if err != nil {
		t.Fatal(err)
	}

	if len(old) != 1 || old[0].CPUPercent != nil || old[0].MemoryBytes != 200 {
		t.Fatalf("old CPU basis must not mix with host shares: %+v", old)
	}
}

func TestVMCPUInMachTicksIsHidden(t *testing.T) {
	ctx := context.Background()
	database, err := InitDB(ctx, filepath.Join(t.TempDir(), "metrics.db"))
	if err != nil {
		t.Fatal(err)
	}

	defer func() { _ = database.Close() }()
	_, err = database.ExecContext(ctx, `INSERT INTO metric_samples
        (scope, target, ts, cpu_percent, memory_bytes, memory_total, rx_bytes, tx_bytes, cpu_basis, counter_source)
        VALUES ('vm','guest',100,0.5,200,500,0,0,'host','mib'),
               ('host','host',100,40,200,500,0,0,'host','mib')`)
	if err != nil {
		t.Fatal(err)
	}

	guest, err := database.MetricsHistory(ctx, "vm", "guest", 0, 5)
	if err != nil {
		t.Fatal(err)
	}

	if len(guest) != 1 || guest[0].CPUPercent != nil {
		t.Fatalf("VM CPU sampled in Mach ticks must be hidden: %+v", guest)
	}

	host, err := database.MetricsHistory(ctx, "host", "host", 0, 5)
	if err != nil {
		t.Fatal(err)
	}

	if len(host) != 1 || host[0].CPUPercent == nil || *host[0].CPUPercent != 40 {
		t.Fatalf("host CPU must stay visible: %+v", host)
	}

	value := 25.0
	if err := database.SaveMetrics(ctx, []MetricSample{{Scope: "vm", Target: "guest", Point: metrics.Point{TS: 200, CPUPercent: &value}}}, 0); err != nil {
		t.Fatal(err)
	}

	current, err := database.MetricsHistory(ctx, "vm", "guest", 150, 5)
	if err != nil {
		t.Fatal(err)
	}

	if len(current) != 1 || current[0].CPUPercent == nil || *current[0].CPUPercent != 25 {
		t.Fatalf("new VM CPU samples must be visible: %+v", current)
	}
}
