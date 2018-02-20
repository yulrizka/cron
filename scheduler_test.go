package cron

import (
	"context"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestScheduler_check(t *testing.T) {
	now := time.Date(2000, 01, 01, 01, 01, 10, 0, time.UTC)

	// there are 3 entries
	entry1, err := Parse("01 01 01 01 *", time.UTC, "ENTRY_1")
	if err != nil {
		t.Fatal(err)
	}
	entry2, err := Parse("01 01 01 01 *", time.UTC, "ENTRY_2")
	if err != nil {
		t.Fatal(err)
	}
	entry3, err := Parse("02 01 01 01 *", time.UTC, "ENTRY_2") // this one should not be triggered
	if err != nil {
		t.Fatal(err)
	}

	// entry one already triggered
	event1 := Event{Entry: entry1, Time: now}

	ctx := context.Background()
	store := MemStore{}
	store.AddEntry(ctx, entry1)
	store.AddEntry(ctx, entry2)
	store.AddEntry(ctx, entry3)
	store.AddEvent(ctx, event1)

	// there are 2 scheduler
	var triggered1 []string
	handler1 := func(name string) {
		triggered1 = append(triggered1, name)
	}
	scheduler1 := NewScheduler(handler1, &store)

	var triggered2 []string
	handler2 := func(name string) {
		triggered2 = append(triggered2, name)
	}
	scheduler2 := NewScheduler(handler2, &store)

	// scheduler1 run before2. scheduler1 should be the only one who triggers events
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := scheduler1.check(ctx, now)
		if err != nil {
			t.Errorf("scheduler1 got error: %v", err)
		}

	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(10 * time.Millisecond)
		err := scheduler2.check(ctx, now)
		if err != nil {
			t.Errorf("scheduler2 got error: %v", err)
		}

	}()

	wg.Wait()
	if got, want := len(store.events), 2; got != want {
		t.Errorf("got %d events want %d", got, want)
	}
	if got, want := store.events[0].Entry.Name, "ENTRY_1"; got != want {
		t.Errorf("got entry 1 name %q want %q", got, want)
	}
	if got, want := store.events[1].Entry.Name, "ENTRY_2"; got != want {
		t.Errorf("got entry 2 name %q want %q", got, want)
	}

	if got, want := triggered1, []string{"ENTRY_2"}; !reflect.DeepEqual(got, want) {
		t.Errorf("got triggered1 %v want %v", got, want)

	}
	if got, want := len(triggered2), 0; !reflect.DeepEqual(got, want) {
		t.Errorf("got length triggered2 %d want %d", got, want)
	}
}
