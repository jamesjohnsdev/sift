// Package storage is the local SQLite cache: the source of truth for
// offline reads and the default (non-"srv:") search, keyed by
// provider.AccountID so multiple accounts share one database file.
package storage

import (
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	for _, pragma := range []string{"PRAGMA foreign_keys = ON", "PRAGMA journal_mode = WAL"} {
		if _, err := db.Exec(pragma); err != nil {
			return nil, errors.Join(fmt.Errorf("set pragma: %w", err), db.Close())
		}
	}

	for _, stmt := range schema {
		if _, err := db.Exec(stmt); err != nil {
			return nil, errors.Join(fmt.Errorf("apply schema: %w", err), db.Close())
		}
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

// rollback is deferred after BeginTx. sql.ErrTxDone from calling it after a
// successful Commit is expected and not a real error.
func rollback(tx *sql.Tx) error {
	if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
		return err
	}
	return nil
}
