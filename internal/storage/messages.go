package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

// UpsertMessages replaces the given messages, their tag associations,
// attachments, and search index entries in a single transaction.
func (s *Store) UpsertMessages(ctx context.Context, account provider.AccountID, msgs []provider.Message) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, m := range msgs {
		if err := upsertMessage(ctx, tx, account, m); err != nil {
			return fmt.Errorf("upsert message %s: %w", m.ID, err)
		}
	}
	return tx.Commit()
}

func upsertMessage(ctx context.Context, tx *sql.Tx, account provider.AccountID, m provider.Message) error {
	to, err := json.Marshal(m.To)
	if err != nil {
		return err
	}
	cc, err := json.Marshal(m.Cc)
	if err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO messages (account_id, id, thread_id, from_addr, to_addrs, cc_addrs, subject, date, snippet, body_text, body_html)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (account_id, id) DO UPDATE SET
			thread_id = excluded.thread_id, from_addr = excluded.from_addr,
			to_addrs = excluded.to_addrs, cc_addrs = excluded.cc_addrs,
			subject = excluded.subject, date = excluded.date, snippet = excluded.snippet,
			body_text = excluded.body_text, body_html = excluded.body_html`,
		account, m.ID, m.ThreadID, m.From, string(to), string(cc),
		m.Subject, m.Date.Unix(), m.Snippet, m.BodyText, m.BodyHTML,
	); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM message_tags WHERE account_id = ? AND message_id = ?`, account, m.ID); err != nil {
		return err
	}
	for _, tagID := range m.Tags {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO message_tags (account_id, message_id, tag_id) VALUES (?, ?, ?)`,
			account, m.ID, tagID); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM attachments WHERE account_id = ? AND message_id = ?`, account, m.ID); err != nil {
		return err
	}
	for _, a := range m.Attachments {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO attachments (account_id, message_id, id, filename, mime_type, size)
			VALUES (?, ?, ?, ?, ?, ?)`,
			account, m.ID, a.ID, a.Filename, a.MIMEType, a.Size); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM messages_fts WHERE account_id = ? AND message_id = ?`, account, m.ID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO messages_fts (subject, body_text, from_addr, account_id, message_id)
		VALUES (?, ?, ?, ?, ?)`,
		m.Subject, m.BodyText, m.From, account, m.ID); err != nil {
		return err
	}

	return nil
}

// Messages returns a page of messages carrying tag, newest first. Pass a
// zero before to fetch the most recent page; pass the oldest date seen on
// the previous page to fetch the next one.
func (s *Store) Messages(ctx context.Context, account provider.AccountID, tag provider.TagID, limit int, before time.Time) ([]provider.Message, error) {
	query := `
		SELECT m.id, m.thread_id, m.from_addr, m.to_addrs, m.cc_addrs, m.subject, m.date, m.snippet, m.body_text, m.body_html
		FROM messages m
		JOIN message_tags mt ON mt.account_id = m.account_id AND mt.message_id = m.id
		WHERE m.account_id = ? AND mt.tag_id = ?`
	args := []any{account, tag}
	if !before.IsZero() {
		query += ` AND m.date < ?`
		args = append(args, before.Unix())
	}
	query += ` ORDER BY m.date DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	msgs, err := scanMessages(rows)
	if err != nil {
		return nil, err
	}
	if err := s.attachTags(ctx, account, msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

func (s *Store) Message(ctx context.Context, account provider.AccountID, id provider.MessageID) (*provider.Message, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, thread_id, from_addr, to_addrs, cc_addrs, subject, date, snippet, body_text, body_html
		FROM messages WHERE account_id = ? AND id = ?`, account, id)

	m, err := scanMessage(row.Scan)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get message: %w", err)
	}

	msgs := []provider.Message{m}
	if err := s.attachTags(ctx, account, msgs); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, filename, mime_type, size FROM attachments WHERE account_id = ? AND message_id = ?`, account, id)
	if err != nil {
		return nil, fmt.Errorf("list attachments: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var a provider.Attachment
		if err := rows.Scan(&a.ID, &a.Filename, &a.MIMEType, &a.Size); err != nil {
			return nil, fmt.Errorf("scan attachment: %w", err)
		}
		msgs[0].Attachments = append(msgs[0].Attachments, a)
	}

	return &msgs[0], rows.Err()
}

// Search runs the local full-text index (the default, non "srv:" search).
func (s *Store) Search(ctx context.Context, account provider.AccountID, query string, limit int) ([]provider.Message, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT m.id, m.thread_id, m.from_addr, m.to_addrs, m.cc_addrs, m.subject, m.date, m.snippet, m.body_text, m.body_html
		FROM messages_fts f
		JOIN messages m ON m.account_id = f.account_id AND m.id = f.message_id
		WHERE f.account_id = ? AND messages_fts MATCH ?
		ORDER BY rank LIMIT ?`, account, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search messages: %w", err)
	}
	defer rows.Close()

	msgs, err := scanMessages(rows)
	if err != nil {
		return nil, err
	}
	if err := s.attachTags(ctx, account, msgs); err != nil {
		return nil, err
	}
	return msgs, nil
}

func scanMessages(rows *sql.Rows) ([]provider.Message, error) {
	var msgs []provider.Message
	for rows.Next() {
		m, err := scanMessage(rows.Scan)
		if err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func scanMessage(scan func(...any) error) (provider.Message, error) {
	var (
		m        provider.Message
		to, cc   string
		dateUnix int64
	)
	if err := scan(&m.ID, &m.ThreadID, &m.From, &to, &cc, &m.Subject, &dateUnix, &m.Snippet, &m.BodyText, &m.BodyHTML); err != nil {
		return provider.Message{}, err
	}
	m.Date = time.Unix(dateUnix, 0)
	if err := json.Unmarshal([]byte(to), &m.To); err != nil {
		return provider.Message{}, err
	}
	if err := json.Unmarshal([]byte(cc), &m.Cc); err != nil {
		return provider.Message{}, err
	}
	return m, nil
}

// attachTags fills in each message's Tags field with one query rather than
// one per message.
func (s *Store) attachTags(ctx context.Context, account provider.AccountID, msgs []provider.Message) error {
	if len(msgs) == 0 {
		return nil
	}

	ids := make([]any, len(msgs)+1)
	ids[0] = account
	placeholders := ""
	for i, m := range msgs {
		ids[i+1] = m.ID
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT message_id, tag_id FROM message_tags WHERE account_id = ? AND message_id IN (`+placeholders+`)`,
		ids...)
	if err != nil {
		return fmt.Errorf("load message tags: %w", err)
	}
	defer rows.Close()

	byMessage := make(map[provider.MessageID][]provider.TagID)
	for rows.Next() {
		var msgID provider.MessageID
		var tagID provider.TagID
		if err := rows.Scan(&msgID, &tagID); err != nil {
			return fmt.Errorf("scan message tag: %w", err)
		}
		byMessage[msgID] = append(byMessage[msgID], tagID)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	for i := range msgs {
		msgs[i].Tags = byMessage[msgs[i].ID]
	}
	return nil
}
