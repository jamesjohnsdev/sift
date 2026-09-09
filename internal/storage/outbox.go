package storage

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/jamesjohnsdev/sift/internal/provider"
)

type OutboxStatus string

const (
	OutboxPending OutboxStatus = "pending"
	OutboxFailed  OutboxStatus = "failed"
)

type OutboxEntry struct {
	ID        string
	Draft     provider.Draft
	Attempts  int
	LastError string
}

// outboxDraft is the JSON-serializable form of provider.Draft: attachment
// content is read into memory once at enqueue time, since io.Reader itself
// can't round-trip through storage.
type outboxDraft struct {
	To, Cc, Bcc  []string
	Subject      string
	MarkdownBody string
	InReplyTo    provider.MessageID
	Attachments  []outboxAttachment
}

type outboxAttachment struct {
	Filename string
	MIMEType string
	Content  []byte
}

func (s *Store) Enqueue(ctx context.Context, account provider.AccountID, draft provider.Draft) (string, error) {
	od := outboxDraft{
		To: draft.To, Cc: draft.Cc, Bcc: draft.Bcc,
		Subject:      draft.Subject,
		MarkdownBody: draft.MarkdownBody,
		InReplyTo:    draft.InReplyTo,
	}
	for _, a := range draft.Attachments {
		content, err := io.ReadAll(a.Content)
		if err != nil {
			return "", fmt.Errorf("read attachment %s: %w", a.Filename, err)
		}
		od.Attachments = append(od.Attachments, outboxAttachment{
			Filename: a.Filename, MIMEType: a.MIMEType, Content: content,
		})
	}

	payload, err := json.Marshal(od)
	if err != nil {
		return "", fmt.Errorf("marshal draft: %w", err)
	}

	id := uuid.NewString()
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO outbox (id, account_id, draft_json, status, attempts, created_at)
		VALUES (?, ?, ?, ?, 0, ?)`,
		id, account, string(payload), OutboxPending, time.Now().Unix())
	if err != nil {
		return "", fmt.Errorf("enqueue draft: %w", err)
	}
	return id, nil
}

// Pending returns queued sends still worth retrying, oldest first.
func (s *Store) Pending(ctx context.Context, account provider.AccountID) (_ []OutboxEntry, err error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, draft_json, attempts, last_error FROM outbox
		WHERE account_id = ? AND status = ? ORDER BY created_at`, account, OutboxPending)
	if err != nil {
		return nil, fmt.Errorf("list outbox: %w", err)
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	var entries []OutboxEntry
	for rows.Next() {
		var (
			e       OutboxEntry
			payload string
		)
		if err := rows.Scan(&e.ID, &payload, &e.Attempts, &e.LastError); err != nil {
			return nil, fmt.Errorf("scan outbox entry: %w", err)
		}

		var od outboxDraft
		if err := json.Unmarshal([]byte(payload), &od); err != nil {
			return nil, fmt.Errorf("unmarshal draft: %w", err)
		}
		e.Draft = draftFromOutbox(od)
		entries = append(entries, e)
	}
	return entries, rows.Err()
}

func draftFromOutbox(od outboxDraft) provider.Draft {
	d := provider.Draft{
		To: od.To, Cc: od.Cc, Bcc: od.Bcc,
		Subject:      od.Subject,
		MarkdownBody: od.MarkdownBody,
		InReplyTo:    od.InReplyTo,
	}
	for _, a := range od.Attachments {
		d.Attachments = append(d.Attachments, provider.DraftAttachment{
			Filename: a.Filename, MIMEType: a.MIMEType, Content: bytes.NewReader(a.Content),
		})
	}
	return d
}

func (s *Store) MarkSent(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM outbox WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("mark sent: %w", err)
	}
	return nil
}

// MarkFailed records a failed send attempt. Once attempts reaches
// maxRetries the entry is marked failed and Pending stops returning it,
// surfacing as the toast the outbox retry policy calls for.
func (s *Store) MarkFailed(ctx context.Context, id string, sendErr error, maxRetries int) error {
	var attempts int
	err := s.db.QueryRowContext(ctx, `SELECT attempts FROM outbox WHERE id = ?`, id).Scan(&attempts)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get attempts: %w", err)
	}

	attempts++
	status := OutboxPending
	if attempts >= maxRetries {
		status = OutboxFailed
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE outbox SET attempts = ?, status = ?, last_error = ? WHERE id = ?`,
		attempts, status, sendErr.Error(), id)
	if err != nil {
		return fmt.Errorf("mark failed: %w", err)
	}
	return nil
}
