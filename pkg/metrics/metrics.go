package metrics

import "math"

type Counters struct {
	Busy, Total, Memory, Capacity, Identity, RX, TX uint64
}

type Point struct {
	TS          int64    `json:"ts"`
	CPUPercent  *float64 `json:"cpu_percent" extensions:"x-nullable=true"`
	MemoryBytes uint64   `json:"memory_bytes"`
	MemoryTotal uint64   `json:"memory_total"`
	RXBytes     uint64   `json:"rx_bytes"`
	TXBytes     uint64   `json:"tx_bytes"`
	RXBytesPS   *float64 `json:"rx_bytes_ps" extensions:"x-nullable=true"`
	TXBytesPS   *float64 `json:"tx_bytes_ps" extensions:"x-nullable=true"`
}

func Rate(current, previous uint64, seconds float64) *float64 {
	if seconds <= 0 || current < previous {
		return nil
	}

	value := float64(current-previous) / seconds
	return &value
}

func CPU(current, previous Counters, seconds float64, cpus int, process bool) *float64 {
	if current.Identity != previous.Identity {
		return nil
	}

	denominator := float64(current.Total) - float64(previous.Total)
	if process {
		denominator = seconds * 1e9 * float64(cpus)
	}

	if denominator <= 0 || current.Busy < previous.Busy {
		return nil
	}

	value := math.Min(100, 100*float64(current.Busy-previous.Busy)/denominator)
	return &value
}
