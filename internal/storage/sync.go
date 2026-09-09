package storage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

// Cursor returns the last saved sync cursor for account, or "" if none.
func (s *Store) Cursor(ctx context.Context, account provider.AccountID) (provider.Cursor, error) {
	var cursor string
	err := s.db.QueryRowContext(ctx,
		`SELECT cursor FROM sync_state WHERE account_id = ?`, account).Scan(&cursor)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("get cursor: %w", err)
	}
	return provider.Cursor(cursor), nil
}

func (s *Store) SetCursor(ctx context.Context, account provider.AccountID, cursor provider.Cursor) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sync_state (account_id, cursor) VALUES (?, ?)
		ON CONFLICT (account_id) DO UPDATE SET cursor = excluded.cursor`,
		account, cursor)
	if err != nil {
		return fmt.Errorf("set cursor: %w", err)
	}
	return nil
}
