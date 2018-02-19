package cron

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// ErrorCh contain error that can not be passed as return value. This gives flexibility to the user to handle err.
// For example if user are using custom logger. If user do not read the channel that error will be silently ignored
var ErrorCh = make(chan error, 1)

func log(err error) {
	select {
	case ErrorCh <- err:
	default:
	}
}

// event is record of executed entry
type Event struct {
	Entry Entry
	Time  time.Time
}

type Store interface {
	// Initialize the store
	Initialize(ctx context.Context) error
	// Lock the store from external read or write
	Lock(ctx context.Context) error
	// Unlock the store
	UnLock(ctx context.Context) error
	// GetEntries retrieve only active entries
	GetEntries(ctx context.Context) ([]Entry, error)
	// AddEntry to the store
	AddEntry(ctx context.Context, entry Entry) error
	// DeleteEntry from the store
	DeleteEntry(ctx context.Context, entry Entry) error
	//WriteEvent which is triggered cron entry
	AddEvent(ctx context.Context, e Event) error
	// GetEvents on [from, to)
	GetEvents(ctx context.Context, from, to time.Time) ([]Event, error)
}

type handler func(name string)

type Scheduler struct {
	handler handler
	store   Store
}

func NewScheduler(handlerFn handler, store Store) *Scheduler {
	s := &Scheduler{
		handler: handlerFn,
		store:   store,
	}

	return s
}

func (s *Scheduler) Run(ctx context.Context) error {
	err := s.store.Initialize(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize store: %v", err)
	}

	// align with next minute
	now := time.Now()
	nextRun := time.Now().Truncate(time.Minute).Add(time.Minute)
	delay := nextRun.Sub(now)
	time.Sleep(delay)

	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ctx.Done():
			ticker.Stop()
			return nil
		case t := <-ticker.C:
			if err := s.check(ctx, t); err != nil {
				log(fmt.Errorf("failed to do check on %s: %v", t, err))
			}
		}
	}

	return nil
}

func (s *Scheduler) check(ctx context.Context, on time.Time) error {
	if s.store == nil {
		return errors.New("empty store")
	}
	err := s.store.Lock(ctx)
	if err != nil {
		return fmt.Errorf("locking store failed: %v", err)
	}
	defer s.store.UnLock(ctx)

	entries, err := s.store.GetEntries(ctx)
	if err != nil {
		return fmt.Errorf("failed to get entries: %v", err)
	}
	until := on.Add(time.Minute)
	events, err := s.store.GetEvents(ctx, on, until)
	if err != nil {
		return fmt.Errorf("failed to get events: %v", err)
	}

	mapTriggeredEvents := make(map[string]struct{})
	timestampLayout := "2006-01-02-15-04"
	for _, e := range events {
		if e.Entry.Name == "" {
			log(fmt.Errorf("got empty name for an event entry %+v", e.Entry))
			continue
		}
		key := e.Entry.Name + "|" + e.Time.Format(timestampLayout)
		mapTriggeredEvents[key] = struct{}{}
	}

	// for each entries, figure which matched and not triggered yet
	onTimestamp := on.Format(timestampLayout)
	for _, e := range entries {
		if e.Name == "" {
			log(fmt.Errorf("got empty name for an event entry %+v", e))
			continue
		}

		if !e.Match(on) {
			continue
		}

		key := e.Name + "|" + onTimestamp
		if _, ok := mapTriggeredEvents[key]; !ok {
			event := Event{
				Entry: e,
				Time:  on,
			}
			if err := s.store.AddEvent(ctx, event); err != nil {
				log(fmt.Errorf("failed to store event: %v", err))
				continue
			}

			go s.handler(e.Name)
		}
	}

	return nil
}
