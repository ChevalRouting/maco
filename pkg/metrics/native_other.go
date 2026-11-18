//go:build !cgo

package metrics

import "fmt"

func ReadHost() (Counters, error) {
	return Counters{}, fmt.Errorf("native metrics require macOS and cgo")
}

func ReadProcess(_ int) (Counters, error) {
	return Counters{}, fmt.Errorf("native metrics require macOS and cgo")
}

func ReadNetwork() (map[string]Counters, error) {
	return nil, fmt.Errorf("native metrics require macOS and cgo")
}
