package cron

import (
	"context"
	"sync"
	"time"
)

type MemStore struct {
	entries []Entry
	events  []Event
	sync.Mutex
}

func (m *MemStore) Initialize(ctx context.Context) error {
	return nil
}

func (m *MemStore) Lock(ctx context.Context) error {
	m.Mutex.Lock()
	return nil
}

func (m *MemStore) UnLock(ctx context.Context) error {
	m.Mutex.Unlock()
	return nil
}

func (m *MemStore) GetEntries(ctx context.Context) ([]Entry, error) {
	return m.entries, nil
}

func (m *MemStore) AddEntry(ctx context.Context, entry Entry) error {
	m.entries = append(m.entries, entry)
	return nil
}

func (m *MemStore) DeleteEntry(ctx context.Context, entry Entry) error {
	var new []Entry
	for _, v := range m.entries {
		if v.expression == entry.expression && v.Name == entry.Name {
			continue
		}
		new = append(new, v)
	}
	m.entries = new
	return nil
}

func (m *MemStore) AddEvent(ctx context.Context, e Event) error {
	m.events = append(m.events, e)
	return nil
}

func (m *MemStore) GetEvents(ctx context.Context, from, to time.Time) ([]Event, error) {
	var ret []Event
	for _, v := range m.events {
		if (v.Time.Equal(from) || v.Time.After(from)) && v.Time.Before(to) {
			ret = append(ret, v)
		}
	}
	return ret, nil
}
