package storage

// schema is applied statement-by-statement on Open, each guarded with
// IF NOT EXISTS so it's safe to run against an existing database.
var schema = []string{
	`CREATE TABLE IF NOT EXISTS accounts (
		id    TEXT PRIMARY KEY,
		kind  TEXT NOT NULL,
		email TEXT NOT NULL
	)`,

	`CREATE TABLE IF NOT EXISTS tags (
		account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
		id         TEXT NOT NULL,
		name       TEXT NOT NULL,
		special    INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (account_id, id)
	)`,

	`CREATE TABLE IF NOT EXISTS messages (
		account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
		id         TEXT NOT NULL,
		thread_id  TEXT NOT NULL,
		from_addr  TEXT NOT NULL,
		to_addrs   TEXT NOT NULL,
		cc_addrs   TEXT NOT NULL,
		subject    TEXT NOT NULL,
		date       INTEGER NOT NULL,
		snippet    TEXT NOT NULL,
		body_text  TEXT NOT NULL,
		body_html  TEXT NOT NULL,
		PRIMARY KEY (account_id, id)
	)`,
	`CREATE INDEX IF NOT EXISTS messages_date_idx ON messages(account_id, date)`,

	`CREATE TABLE IF NOT EXISTS message_tags (
		account_id TEXT NOT NULL,
		message_id TEXT NOT NULL,
		tag_id     TEXT NOT NULL,
		PRIMARY KEY (account_id, message_id, tag_id),
		FOREIGN KEY (account_id, message_id) REFERENCES messages(account_id, id) ON DELETE CASCADE
	)`,
	`CREATE INDEX IF NOT EXISTS message_tags_tag_idx ON message_tags(account_id, tag_id)`,

	`CREATE TABLE IF NOT EXISTS attachments (
		account_id TEXT NOT NULL,
		message_id TEXT NOT NULL,
		id         TEXT NOT NULL,
		filename   TEXT NOT NULL,
		mime_type  TEXT NOT NULL,
		size       INTEGER NOT NULL,
		PRIMARY KEY (account_id, message_id, id)
	)`,

	`CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
		subject,
		body_text,
		from_addr,
		account_id UNINDEXED,
		message_id UNINDEXED
	)`,

	`CREATE TABLE IF NOT EXISTS sync_state (
		account_id TEXT PRIMARY KEY,
		cursor     TEXT NOT NULL DEFAULT ''
	)`,

	`CREATE TABLE IF NOT EXISTS outbox (
		id          TEXT PRIMARY KEY,
		account_id  TEXT NOT NULL,
		draft_json  TEXT NOT NULL,
		status      TEXT NOT NULL DEFAULT 'pending',
		attempts    INTEGER NOT NULL DEFAULT 0,
		last_error  TEXT NOT NULL DEFAULT '',
		created_at  INTEGER NOT NULL
	)`,
	`CREATE INDEX IF NOT EXISTS outbox_status_idx ON outbox(status)`,
}
