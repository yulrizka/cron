package cron

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

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

	// TODO: check database

	return store, nil
}

// Lock the table so that no other session can read or write Entries and Triggered table
func (s *SqlStore) Lock(ctx context.Context) error {
	if s.locked || s.tx != nil {
		return errors.New("already locked or transaction exists")
	}

	// we use transaction because it's guarantee to give the same connection from SQL pool
	var err error
	txOptions := &sql.TxOptions{
		Isolation: sql.LevelSerializable, // make sure that none is reading and writing to the table we lock
	}
	s.tx, err = s.db.BeginTx(ctx, txOptions)
	if err != nil {
		return fmt.Errorf("failed to create transaction: %v", err)
	}

	_, err = s.tx.ExecContext(ctx, "LOCK TABLE ? WRITE, ? Write", EntriesTable, EventsTable)

	return err
}

func (s *SqlStore) GetEntries(ctx context.Context) ([]Entry, error) {
	if ctx == nil {
		return nil, errors.New("empty context")
	}

	entries := make([]Entry, 0)
	rows, err := s.tx.QueryContext(ctx, "SELECT expression, name, location FROM ? WHERE active=1", EntriesTable)
	if err != nil {
		return nil, fmt.Errorf("failed to query entries from DB: %v", err)
	}

	for rows.Next() {
		var expression, location, name string
		if err := rows.Scan(&expression, &location, &name); err != nil {
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
		entries = append(entries, entry)
	}

	return entries, nil
}

func (s *SqlStore) WriteTriggered(ctx context.Context, e Entry, t time.Time) error {
	if ctx == nil {
		return errors.New("empty context")
	}
	panic("implement me")
}

func (s *SqlStore) UnLock(ctx context.Context) error {
	if ctx == nil {
		return errors.New("empty context")
	}
	if !s.locked || s.tx == nil {
		return errors.New("not locked or transaction not exists")
	}

	_, err := s.tx.ExecContext(ctx, "UNLOCK TABLES")

	return err
}
