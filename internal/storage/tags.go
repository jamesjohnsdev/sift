package storage

import (
	"context"
	"fmt"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

func (s *Store) UpsertTags(ctx context.Context, account provider.AccountID, tags []provider.Tag) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO tags (account_id, id, name, special) VALUES (?, ?, ?, ?)
		ON CONFLICT (account_id, id) DO UPDATE SET name = excluded.name, special = excluded.special`)
	if err != nil {
		return fmt.Errorf("prepare tag upsert: %w", err)
	}
	defer stmt.Close()

	for _, t := range tags {
		if _, err := stmt.ExecContext(ctx, account, t.ID, t.Name, t.Special); err != nil {
			return fmt.Errorf("upsert tag %s: %w", t.ID, err)
		}
	}
	return tx.Commit()
}

func (s *Store) Tags(ctx context.Context, account provider.AccountID) ([]provider.Tag, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, special FROM tags WHERE account_id = ? ORDER BY name`, account)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var tags []provider.Tag
	for rows.Next() {
		var t provider.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Special); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}
