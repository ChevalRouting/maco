package jobs

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/m-vinc/maco/pkg/config"
	"github.com/m-vinc/maco/pkg/engine"
)

func testService(t *testing.T) *Service {
	t.Helper()
	paths, err := config.Resolve(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if err := paths.EnsureDirs(); err != nil {
		t.Fatal(err)
	}

	service, err := New(engine.New(paths))
	if err != nil {
		t.Fatal(err)
	}

	return service
}

func insertJob(t *testing.T, s *Service, id, state string, created time.Time) {
	t.Helper()
	if _, err := s.db.Exec(`INSERT INTO jobs (id, action, target, state, created_at) VALUES (?, 'vm.start', ?, ?, ?)`, id, id, state, created.UnixMilli()); err != nil {
		t.Fatal(err)
	}
}

func TestClaimHoldsUntilExpiry(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	first, err := s.claim(ctx, "lease", "1", time.Minute)
	if err != nil || !first {
		t.Fatalf("first claim: %v %v", first, err)
	}

	second, err := s.claim(ctx, "lease", "1", time.Minute)
	if err != nil || second {
		t.Fatalf("held claim was taken again: %v %v", second, err)
	}

	if _, err := s.db.Exec(`UPDATE job_claims SET expires_at = 0 WHERE key = 'lease'`); err != nil {
		t.Fatal(err)
	}

	value, err := s.claimValue(ctx, "lease")
	if err != nil || value != "" {
		t.Fatalf("expired claim still readable: %q %v", value, err)
	}

	third, err := s.claim(ctx, "lease", "1", time.Minute)
	if err != nil || !third {
		t.Fatalf("expired claim was not reclaimed: %v %v", third, err)
	}
}

func TestReconcileOrphansFailsLostWork(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	insertJob(t, s, "lost", "pending", time.Now().Add(-2*time.Minute))
	insertJob(t, s, "fresh", "pending", time.Now())
	queued, err := s.Submit(ctx, Payload{Action: "vm.start", Target: "queued"})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := s.db.Exec(`UPDATE jobs SET created_at = ? WHERE id = ?`, time.Now().Add(-2*time.Minute).UnixMilli(), queued.ID); err != nil {
		t.Fatal(err)
	}

	s.reconcileOrphans(ctx)
	lost, err := s.Get(ctx, "lost")
	if err != nil || lost.State != Failed || !strings.Contains(lost.Error, "before it started") || len(lost.Logs) != 1 {
		t.Fatalf("lost job not failed: %v %+v", err, lost)
	}

	for _, id := range []string{"fresh", queued.ID} {
		job, err := s.Get(ctx, id)
		if err != nil || job.State != Pending {
			t.Fatalf("job %s should stay pending: %v %+v", id, err, job)
		}
	}
}

func TestTrimHistoryKeepsActiveJobs(t *testing.T) {
	s := testService(t)
	ctx := context.Background()
	start := time.Now().Add(-time.Hour)
	insertJob(t, s, "oldest-pending", "pending", start.Add(-time.Hour))
	for number := range historyLimit + 5 {
		insertJob(t, s, fmt.Sprintf("done-%d", number), "succeeded", start.Add(time.Duration(number)*time.Millisecond))
	}

	s.trimHistory(ctx)
	var finished, pending int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM jobs WHERE state = 'succeeded'`).Scan(&finished); err != nil {
		t.Fatal(err)
	}

	if err := s.db.QueryRow(`SELECT COUNT(*) FROM jobs WHERE state = 'pending'`).Scan(&pending); err != nil {
		t.Fatal(err)
	}

	if finished != historyLimit || pending != 1 {
		t.Fatalf("finished %d pending %d", finished, pending)
	}
}
