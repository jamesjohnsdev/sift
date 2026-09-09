package storage

import (
	"context"
	"fmt"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

type Account struct {
	ID    provider.AccountID
	Kind  provider.Kind
	Email string
}

func (s *Store) UpsertAccount(ctx context.Context, a Account) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (id, kind, email) VALUES (?, ?, ?)
		ON CONFLICT (id) DO UPDATE SET kind = excluded.kind, email = excluded.email`,
		a.ID, a.Kind, a.Email)
	if err != nil {
		return fmt.Errorf("upsert account: %w", err)
	}
	return nil
}

func (s *Store) Accounts(ctx context.Context) ([]Account, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, kind, email FROM accounts ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Kind, &a.Email); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}
