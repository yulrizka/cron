package cron

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

var (
	// EntriesTable in SQL table that store cron entries
	EntriesTable = "_entries"
	// EventsTable is SQL table that store executed entries
	EventsTable = "_events"
)

type SqlStore struct {
	db     *sql.DB
	tx     *sql.Tx
	locked bool
}

func NewSQLStore(db *sql.DB) (*SqlStore, error) {
	store := &SqlStore{db: db}

	return store, nil
}

// Initialize the sql tables if not present
func (s *SqlStore) Initialize(ctx context.Context) error {
	// For now this is enough with assumption that this table is going to be stable.
	// If in the future we need to migrate this we can introduce `_version` table for doing db migration
	// right now, absence of that table marks that this is the initial version

	// create entries table
	query := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  expression varchar(255) NOT NULL,
  location varchar(255) NOT NULL,
  name varchar(255) NOT NULL,
  meta varchar(1024) DEFAULT NULL,
  active tinyint(1) DEFAULT '1',
  PRIMARY KEY (expression,location,name)
)
`, EntriesTable)
	_, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed creating entries table: %v", err)
	}

	// create events table
	query = fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
  expression varchar(255) NOT NULL,
  location varchar(255) NOT NULL,
  name varchar(255) NOT NULL,
  meta varchar(1024) DEFAULT NULL,
  triggered_at timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (expression,location,name,triggered_at)
)`, EventsTable)
	_, err = s.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed creating events table: %v", err)
	}

	return nil
}

// Lock the table so that no other session can read or write Entries and Triggered table
func (s *SqlStore) Lock(ctx context.Context) error {
	if s.locked || s.tx != nil {
		return errors.New("already locked or transaction exists")
	}

	// we use transaction because it guaranteed to give the same connection from SQL pool
	var err error
	txOptions := &sql.TxOptions{
		Isolation: sql.LevelSerializable, // make sure that none is reading and writing to the table we lock
	}
	s.tx, err = s.db.BeginTx(ctx, txOptions)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %v", err)
	}

	_, err = s.tx.ExecContext(ctx, fmt.Sprintf("LOCK TABLE `%s` WRITE, `%s` WRITE", EntriesTable, EventsTable))
	if err != nil {
		return err
	}
	s.locked = true

	return nil
}

func (s *SqlStore) UnLock(ctx context.Context) error {
	if !s.locked || s.tx == nil {
		return errors.New("not locked or transaction not exists")
	}
	_, err := s.tx.ExecContext(ctx, "UNLOCK TABLES")
	if err != nil {
		return err
	}
	s.locked = false

	return nil
}

func (s *SqlStore) AddEntry(ctx context.Context, entry Entry) error {
	if entry.expression == "" {
		return errors.New("got empty expression")
	}
	query := "REPLACE INTO " + EntriesTable + " (expression, location, name, meta) VALUES (?, ?, ?, ?)"
	_, err := s.tx.ExecContext(ctx, query, entry.expression, entry.Location.String(), entry.Name, entry.Meta)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}

	return nil
}

func (s *SqlStore) GetEntries(ctx context.Context) ([]Entry, error) {
	entries := make([]Entry, 0)
	query := "SELECT expression, location, name, meta FROM " + EntriesTable + " WHERE active=1"
	rows, err := s.tx.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries from DB: %v", err)
	}

	for rows.Next() {
		var expression, location, name string
		var meta sql.NullString
		if err := rows.Scan(&expression, &location, &name, &meta); err != nil {
			return nil, fmt.Errorf("failed reading a row: %v", err)
		}
		loc, err := time.LoadLocation(location)
		if err != nil {
			return nil, fmt.Errorf("failed to load location %q: %v", location, err)
		}
		entry, err := Parse(expression, loc, name)
		if err != nil {
			return nil, fmt.Errorf("failed to parse expression:%q loc:%q name:%q: %v", expression, loc, name, err)
		}
		entry.Meta = meta.String

		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *SqlStore) DeleteEntry(ctx context.Context, entry Entry) error {
	query := "DELETE FROM " + EntriesTable + " WHERE expression=? AND location=? AND name=?"
	_, err := s.tx.ExecContext(ctx, query, entry.expression, entry.Location.String(), entry.Name)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}

	return nil
}

func (s *SqlStore) AddEvent(ctx context.Context, e Event) error {
	query := "REPLACE INTO " + EventsTable + " (expression, location, name, triggered_at, meta) VALUES (?, ?, ?, ?, ?)"
	expression := e.Entry.expression
	location := e.Entry.Location.String()
	name := e.Entry.Name
	_, err := s.tx.ExecContext(ctx, query, expression, location, name, e.Time, e.Entry.Meta)
	if err != nil {
		return fmt.Errorf("failed to execute query: %v", err)
	}

	return nil
}

func (s *SqlStore) GetEvents(ctx context.Context, from, to time.Time) ([]Event, error) {
	query := `SELECT expression, location, name, meta, triggered_at from ` + EventsTable + ` WHERE triggered_at >= ? AND triggered_at <= ?`
	rows, err := s.tx.QueryContext(ctx, query, from, to)
	if err != nil {
		return nil, fmt.Errorf("failed querying database: %v", err)
	}

	var events []Event
	for rows.Next() {
		var ev Event
		var expression, location, name string
		var meta sql.NullString
		var triggeredAt time.Time

		if err := rows.Scan(&expression, &location, &name, &meta, &triggeredAt); err != nil {
			return nil, fmt.Errorf("failed reading a row: %v", err)
		}

		loc, err := time.LoadLocation(location)
		if err != nil {
			return nil, fmt.Errorf("failed to load location %q: %v", location, err)
		}
		entry, err := Parse(expression, loc, name)
		if err != nil {
			return nil, fmt.Errorf("failed to load entry expression:%q loc:%q name:%q: %v", expression, loc, name, err)
		}
		entry.Meta = meta.String
		ev.Entry = entry
		ev.Time = triggeredAt.In(loc)
		events = append(events, ev)
	}

	return events, nil
}
