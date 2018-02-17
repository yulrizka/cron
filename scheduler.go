package cron

import (
	"context"
	"fmt"
	"time"
)

// event is record of executed entry
type Event struct {
	Entry Entry
	Time  time.Time
}

type Store interface {
	// Initialize the store
	Initialize() error
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
	WriteEvent(ctx context.Context, e Event) error
	// GetEvents inclusive time rage
	GetEvents(ctx context.Context, from, to time.Time) ([]Event, error)
}

type Scheduler struct {
	handler func()
	store   Store
}

func NewScheduler(handlerFn func(), store Store) (*Scheduler, error) {
	s := &Scheduler{
		handler: handlerFn,
		store:   store,
	}

	err := store.Initialize()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize store: %v", err)
	}

	return s, nil
}

func (s *Scheduler) Run(ctx context.Context) error {
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
		case ticker.C:
			// perform check
		}
	}

	// calculate next run to run timer
	// sleep
	// evey minute:
	//   lock data store
	//   defer unlock data store
	//   get active entries
	//   get events
	//   for each active entries:
	//      check if there is active events. no return
	//      write the event to the store
	//      call handler in go routine if not exist yet

	return nil
}
