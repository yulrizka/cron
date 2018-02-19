package cron

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestCron_Memstore(t *testing.T) {
	store := &MemStore{}
	ctx := context.Background()

	err := store.Initialize(ctx)
	if err != nil {
		t.Fatal(err)
	}

	err = store.Lock(ctx)
	if err != nil {
		t.Fatal(err)
	}

	entry, err := Parse("* * * * *", time.UTC, "ENTRY_1")
	if err != nil {
		t.Fatal(err)
	}

	err = store.AddEntry(ctx, entry)
	if err != nil {
		t.Fatal(err)
	}

	entries, err := store.GetEntries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("got entries %d want %d", got, want)
	}
	if got, want := entries[0], entry; !reflect.DeepEqual(got, want) {
		t.Fatalf("got entry %+v want %+v", got, want)
	}

	ev := Event{
		Entry: entry,
		Time:  time.Now(),
	}
	err = store.AddEvent(ctx, ev)
	if err != nil {
		t.Fatal(err)
	}

	events, err := store.GetEvents(ctx, ev.Time, ev.Time.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(events), 1; got != want {
		t.Fatalf("got events %d want %d", got, want)
	}
	if got, want := events[0], ev; !reflect.DeepEqual(got, want) {
		t.Fatalf("got events %+v want %+v", got, want)
	}

	entry2, err := Parse("* * * * *", time.UTC, "ENTRY_2")
	if err != nil {
		t.Fatal(err)
	}
	err = store.AddEntry(ctx, entry2)
	if err != nil {
		t.Fatal(err)
	}

	err = store.DeleteEntry(ctx, entry)
	if err != nil {
		t.Fatal(err)
	}

	entries, err = store.GetEntries(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(entries), 1; got != want {
		t.Fatalf("got entries %d want %d", got, want)
	}
	if got, want := entries[0], entry2; !reflect.DeepEqual(got, want) {
		t.Fatalf("got entry %+v want %+v", got, want)
	}
}
