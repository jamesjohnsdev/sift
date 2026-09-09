package provider

import (
	"io"
	"time"
)

type (
	AccountID    string
	TagID        string
	MessageID    string
	AttachmentID string
)

// SpecialTag marks the reserved tags with behavior beyond a plain label.
// Everything else is freeform, including Outlook/IMAP folder paths
// flattened to "Parent/Child".
type SpecialTag int

const (
	TagNone SpecialTag = iota
	TagInbox
	TagSent
	TagDrafts
	TagTrash
	TagSpam
)

type Tag struct {
	ID      TagID
	Name    string
	Special SpecialTag
}

type Attachment struct {
	ID       AttachmentID
	Filename string
	MIMEType string
	Size     int64
}

type Message struct {
	ID          MessageID
	ThreadID    string
	From        string
	To, Cc      []string
	Subject     string
	Date        time.Time
	Tags        []TagID
	Snippet     string
	BodyText    string
	BodyHTML    string
	Attachments []Attachment
}

// Draft is a message being composed. Body is Markdown; providers convert it
// to HTML (plus a plaintext part) at send time.
type Draft struct {
	To, Cc, Bcc  []string
	Subject      string
	MarkdownBody string
	Attachments  []DraftAttachment
	InReplyTo    MessageID
}

type DraftAttachment struct {
	Filename string
	MIMEType string
	Content  io.Reader
}
