package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"sync"
	"time"

	"github.com/m-vinc/maco/pkg/net/host"
)

type resourceHub struct {
	mu      sync.Mutex
	clients map[chan []string]struct{}
	cancel  context.CancelFunc
}

func (h *resourceHub) broadcast(resources ...string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for client := range h.clients {
		select {
		case client <- resources:
		default:
			select {
			case <-client:
			default:
			}

			select {
			case client <- []string{"all"}:
			default:
			}
		}
	}
}

func (s *Server) subscribeResources() (<-chan []string, func()) {
	h := &s.events
	h.mu.Lock()
	if h.clients == nil {
		h.clients = make(map[chan []string]struct{})
	}

	client := make(chan []string, 32)
	h.clients[client] = struct{}{}
	if h.cancel == nil {
		ctx, cancel := context.WithCancel(context.Background())
		h.cancel = cancel
		go s.observeResources(ctx)
	}

	h.mu.Unlock()
	return client, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		delete(h.clients, client)
		if len(h.clients) == 0 {
			h.cancel()
			h.cancel = nil
		}
	}
}

func (s *Server) observeResources(ctx context.Context) {
	previous := make(map[string][32]byte)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	lastGuestRefresh := time.Time{}
	loaders := map[string]func(context.Context) (any, error){
		"vms": func(ctx context.Context) (any, error) {
			views, err := s.engine.ListVMs(ctx)
			if err == nil && time.Since(lastGuestRefresh) >= 15*time.Second {
				lastGuestRefresh = time.Now()
				for _, view := range views {
					s.events.broadcast("guest-agent:" + view.Manifest.ID)
				}
			}

			return views, err
		},
		"networks":   func(context.Context) (any, error) { return s.engine.ListNetworks() },
		"media":      func(context.Context) (any, error) { return s.engine.ListMedia() },
		"catalog":    func(context.Context) (any, error) { return s.engine.Catalog(), nil },
		"disks":      func(context.Context) (any, error) { return s.engine.ListDisks() },
		"storage":    func(context.Context) (any, error) { return s.engine.StorageStats() },
		"interfaces": func(context.Context) (any, error) { return host.Ports() },
		"usb":        func(ctx context.Context) (any, error) { return s.engine.ListUSBDevices(ctx) },
	}
	for {
		for domain, load := range loaders {
			if ctx.Err() != nil {
				return
			}

			readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			value, err := load(readCtx)
			cancel()
			message := ""
			if err != nil {
				message = err.Error()
			}

			data, marshalErr := json.Marshal(struct {
				Value any
				Error string
			}{value, message})
			if marshalErr != nil {
				continue
			}

			hash := sha256.Sum256(data)
			if old, ok := previous[domain]; !ok || old != hash {
				previous[domain] = hash
				s.events.broadcast(domain)
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
