package jobs

import (
	"context"
	"time"
)

const eventPollInterval = 250 * time.Millisecond

type Event struct {
	ID      int64
	Payload string
}

type Subscription struct {
	events  chan Event
	cancel  context.CancelFunc
	stopped chan struct{}
}

func (s *Subscription) Channel() <-chan Event {
	return s.events
}

func (s *Subscription) Close() error {
	s.cancel()
	<-s.stopped
	return nil
}

func (s *Service) Subscribe(ctx context.Context) (*Subscription, error) {
	var cursor int64
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(id), 0) FROM job_events`).Scan(&cursor); err != nil {
		return nil, err
	}

	pollCtx, cancel := context.WithCancel(context.Background())
	subscription := &Subscription{events: make(chan Event, 64), cancel: cancel, stopped: make(chan struct{})}
	go s.pollEvents(pollCtx, subscription, cursor)
	return subscription, nil
}

func (s *Service) pollEvents(ctx context.Context, subscription *Subscription, cursor int64) {
	defer close(subscription.stopped)
	defer close(subscription.events)
	ticker := time.NewTicker(eventPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}

		events, err := s.eventsAfter(ctx, cursor)
		if err != nil {
			continue
		}

		for _, event := range events {
			select {
			case <-ctx.Done():
				return
			case subscription.events <- event:
				cursor = event.ID
			}
		}
	}
}

func (s *Service) eventsAfter(ctx context.Context, cursor int64) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, payload FROM job_events WHERE id > ? ORDER BY id LIMIT 500`, cursor)
	if err != nil {
		return nil, err
	}

	defer func() { _ = rows.Close() }()
	var events []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.Payload); err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, rows.Err()
}
