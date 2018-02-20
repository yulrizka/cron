package cron

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestCron_MemStore(t *testing.T) {
	store := &MemStore{}
	storeTest(t, store)
}

func TestCron_SQLStore(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	env := func(key, defaultValue string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return defaultValue
	}
	user := env("MYSQL_TEST_USER", "gotest")
	pass := env("MYSQL_TEST_PASS", "secret")
	prot := env("MYSQL_TEST_PROT", "tcp")
	addr := env("MYSQL_TEST_ADDR", "localhost:3306")
	dbname := env("MYSQL_TEST_DBNAME", "gotest")
	netAddr := fmt.Sprintf("%s(%s)", prot, addr)
	dsn := fmt.Sprintf("%s:%s@%s/%s?timeout=30s&parseTime=true", user, pass, netAddr, dbname)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}

	store, err := NewSQLStore(db) // This will check and create necessary table if not exists
	if err != nil {
		t.Fatalf("Failed to initialize MysqlPersister: %v", err)
	}
	storeTest(t, store)
}

func storeTest(t *testing.T, store Store) {
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
	entry.Meta = "META"

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
		Time:  time.Date(2018, 12, 15, 0, 0, 0, 0, time.UTC),
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

	err = store.DeleteEntry(ctx, entry2)
	if err != nil {
		t.Fatal(err)
	}

	err = store.UnLock(ctx)
	if err != nil {
		t.Fatal(err)
	}
}
