package metrics

import "testing"

func TestRatesAndCPUNormalization(t *testing.T) {
	if rate := Rate(300, 100, 2); rate == nil || *rate != 100 {
		t.Fatalf("rate = %v", rate)
	}

	if Rate(10, 100, 2) != nil || Rate(100, 10, 0) != nil {
		t.Fatal("reset and missing elapsed time must have no rate")
	}

	previous := Counters{Busy: 100, Total: 200}
	current := Counters{Busy: 150, Total: 400}
	if cpu := CPU(current, previous, 5, 4, false); cpu == nil || *cpu != 25 {
		t.Fatalf("host CPU = %v", cpu)
	}

	previous = Counters{Busy: 1e9, Identity: 12}
	current = Counters{Busy: 5e9, Identity: 12}
	if cpu := CPU(current, previous, 2, 4, true); cpu == nil || *cpu != 50 {
		t.Fatalf("VM CPU = %v", cpu)
	}

	current.Identity++
	if CPU(current, previous, 2, 4, true) != nil {
		t.Fatal("restarted VM must establish a new baseline")
	}
}
