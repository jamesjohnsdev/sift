package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/mail"
	"strings"
	"testing"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

// decodeRaw extracts and base64url-decodes the "raw" field posted to
// messages.send, returning the parsed RFC 2822 message.
func decodeRaw(t *testing.T, body io.Reader) *mail.Message {
	t.Helper()
	var req struct {
		Raw string `json:"raw"`
	}
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		t.Fatalf("decode request body: %v", err)
	}
	raw, err := base64.RawURLEncoding.DecodeString(req.Raw)
	if err != nil {
		t.Fatalf("base64url-decode raw: %v", err)
	}
	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parse raw message: %v", err)
	}
	return msg
}

// readAlternativeParts walks a multipart/alternative (or /mixed containing
// one) body and returns the text/plain and text/html part contents.
func readAlternativeParts(t *testing.T, msg *mail.Message) (plain, html string, attachments []string) {
	t.Helper()

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse Content-Type: %v", err)
	}
	if !strings.HasPrefix(mediaType, "multipart/") {
		t.Fatalf("Content-Type = %q, want multipart/*", mediaType)
	}
	return readMultipart(t, msg.Body, params["boundary"])
}

func readMultipart(t *testing.T, r io.Reader, boundary string) (plain, html string, attachments []string) {
	t.Helper()
	mr := multipart.NewReader(r, boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("next inner part: %v", err)
		}
		partType, partParams, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("parse inner part Content-Type: %v", err)
		}
		if strings.HasPrefix(partType, "multipart/") {
			p2, h2, a2 := readMultipart(t, part, partParams["boundary"])
			plain, html = p2, h2
			attachments = append(attachments, a2...)
			continue
		}
		data, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("read inner part: %v", err)
		}
		if disp := part.Header.Get("Content-Disposition"); strings.HasPrefix(disp, "attachment") {
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(data)))
			if err != nil {
				t.Fatalf("base64-decode attachment part: %v", err)
			}
			attachments = append(attachments, string(decoded))
			continue
		}
		switch partType {
		case "text/plain":
			plain = string(data)
		case "text/html":
			html = string(data)
		}
	}
	return plain, html, attachments
}

func TestSendWithoutAttachments(t *testing.T) {
	var captured *mail.Message

	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/messages/send" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		captured = decodeRaw(t, r.Body)
		writeJSON(t, w, map[string]string{"id": "sent-1"})
	}))

	draft := provider.Draft{
		To:           []string{"bob@example.com"},
		Subject:      "Q3 roadmap",
		MarkdownBody: "**hi** there",
	}
	if err := p.Send(context.Background(), draft); err != nil {
		t.Fatalf("Send: %v", err)
	}

	if got := captured.Header.Get("To"); got != "bob@example.com" {
		t.Fatalf("To = %q, want bob@example.com", got)
	}
	if got := captured.Header.Get("Subject"); got != "Q3 roadmap" {
		t.Fatalf("Subject = %q, want Q3 roadmap", got)
	}

	plain, html, atts := readAlternativeParts(t, captured)
	if plain != "**hi** there" {
		t.Fatalf("plain part = %q, want raw markdown", plain)
	}
	if !strings.Contains(html, "<strong>hi</strong>") {
		t.Fatalf("html part = %q, want rendered markdown", html)
	}
	if len(atts) != 0 {
		t.Fatalf("attachments = %v, want none", atts)
	}
}

func TestSendWithAttachment(t *testing.T) {
	var captured *mail.Message

	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = decodeRaw(t, r.Body)
		writeJSON(t, w, map[string]string{"id": "sent-2"})
	}))

	draft := provider.Draft{
		To:           []string{"bob@example.com"},
		Subject:      "With attachment",
		MarkdownBody: "see attached",
		Attachments: []provider.DraftAttachment{
			{Filename: "notes.txt", MIMEType: "text/plain", Content: strings.NewReader("attachment content")},
		},
	}
	if err := p.Send(context.Background(), draft); err != nil {
		t.Fatalf("Send: %v", err)
	}

	mediaType, _, err := mime.ParseMediaType(captured.Header.Get("Content-Type"))
	if err != nil {
		t.Fatalf("parse Content-Type: %v", err)
	}
	if mediaType != "multipart/mixed" {
		t.Fatalf("Content-Type = %q, want multipart/mixed", mediaType)
	}

	plain, _, atts := readAlternativeParts(t, captured)
	if plain != "see attached" {
		t.Fatalf("plain part = %q", plain)
	}
	if len(atts) != 1 || atts[0] != "attachment content" {
		t.Fatalf("attachments = %v, want [attachment content]", atts)
	}
}

func TestAttachment(t *testing.T) {
	want := "the attachment bytes"
	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages/m1/attachments/att-1" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeJSON(t, w, attachmentResponse{
			Size: len(want),
			Data: base64.RawURLEncoding.EncodeToString([]byte(want)),
		})
	}))

	rc, err := p.Attachment(context.Background(), "m1", "att-1")
	if err != nil {
		t.Fatalf("Attachment: %v", err)
	}
	defer func() { _ = rc.Close() }()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read attachment: %v", err)
	}
	if string(got) != want {
		t.Fatalf("Attachment content = %q, want %q", got, want)
	}
}
