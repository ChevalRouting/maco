//go:build cgo

package metrics

import (
	"os"
	"testing"
	"time"
)

func TestNativeStatistics(t *testing.T) {
	host, err := ReadHost()
	if err != nil {
		t.Fatal(err)
	}

	if host.Capacity == 0 || host.Memory == 0 || host.Total < host.Busy {
		t.Fatalf("invalid host counters: %+v", host)
	}

	process, err := ReadProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}

	if process.Memory == 0 || process.Identity == 0 {
		t.Fatalf("invalid process counters: %+v", process)
	}

	interfaces, err := ReadNetwork()
	if err != nil {
		t.Fatal(err)
	}

	if _, ok := interfaces["lo0"]; ok {
		t.Fatal("loopback should not be included")
	}

	t.Logf("host used=%d total=%d, process resident=%d, interfaces=%v", host.Memory, host.Capacity, process.Memory, interfaces)
}

func TestProcessCPUIsNanoseconds(t *testing.T) {
	before, err := ReadProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	for time.Since(start) < 500*time.Millisecond {
	}

	after, err := ReadProcess(os.Getpid())
	if err != nil {
		t.Fatal(err)
	}

	busy := float64(after.Busy-before.Busy) / float64(time.Since(start).Nanoseconds())
	if busy < 0.5 || busy > 1.5 {
		t.Fatalf("one busy core measured as %.3f cores", busy)
	}
}
