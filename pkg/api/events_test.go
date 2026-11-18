package api

import "testing"

func TestResourceHubSlowClientResynchronizes(t *testing.T) {
	client := make(chan []string, 1)
	hub := resourceHub{clients: map[chan []string]struct{}{client: {}}}
	hub.broadcast("networks")
	hub.broadcast("media")
	resources := <-client
	if len(resources) != 1 || resources[0] != "all" {
		t.Fatalf("missed change instead of resync: %v", resources)
	}
}
