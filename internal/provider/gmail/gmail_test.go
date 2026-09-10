package gmail

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

func newTestProvider(t *testing.T, handler http.Handler) *Provider {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &Provider{account: "acct-1", client: srv.Client(), baseURL: srv.URL}
}

func writeJSON(t *testing.T, w http.ResponseWriter, v any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		t.Fatalf("writeJSON: %v", err)
	}
}

func TestTags(t *testing.T) {
	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/labels" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]any{
			"labels": []map[string]string{
				{"id": "INBOX", "name": "INBOX", "type": "system"},
				{"id": "UNREAD", "name": "UNREAD", "type": "system"},
				{"id": "Label_1", "name": "Work/Projects", "type": "user"},
			},
		})
	}))

	tags, err := p.Tags(context.Background())
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("Tags = %+v, want 2 (INBOX kept, UNREAD dropped, Work/Projects kept)", tags)
	}
	if tags[0].ID != "INBOX" || tags[0].Special != provider.TagInbox {
		t.Fatalf("Tags[0] = %+v, want special INBOX", tags[0])
	}
	if tags[1].Name != "Work/Projects" || tags[1].Special != provider.TagNone {
		t.Fatalf("Tags[1] = %+v, want freeform Work/Projects", tags[1])
	}
}

func TestMessagesPaginatesAndFetchesDetail(t *testing.T) {
	pages := map[string]any{
		"": map[string]any{
			"messages":      []map[string]string{{"id": "m1"}},
			"nextPageToken": "page2",
		},
		"page2": map[string]any{
			"messages": []map[string]string{{"id": "m2"}},
		},
	}

	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/messages":
			writeJSON(t, w, pages[r.URL.Query().Get("pageToken")])
		case "/messages/m1":
			writeJSON(t, w, plainMessageFixture("m1", "First"))
		case "/messages/m2":
			writeJSON(t, w, plainMessageFixture("m2", "Second"))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	page1, err := p.Messages(context.Background(), "INBOX", "")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if !page1.HasMore || page1.Next != "page2" {
		t.Fatalf("page1 = %+v, want HasMore + Next=page2", page1)
	}
	if len(page1.Messages) != 1 || page1.Messages[0].Subject != "First" {
		t.Fatalf("page1.Messages = %+v", page1.Messages)
	}

	page2, err := p.Messages(context.Background(), "INBOX", page1.Next)
	if err != nil {
		t.Fatalf("Messages(page2): %v", err)
	}
	if page2.HasMore {
		t.Fatalf("page2.HasMore = true, want false (no nextPageToken)")
	}
	if len(page2.Messages) != 1 || page2.Messages[0].Subject != "Second" {
		t.Fatalf("page2.Messages = %+v", page2.Messages)
	}
}

func TestMessageParsesMultipartBodyAndAttachment(t *testing.T) {
	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, map[string]any{
			"id":           "m1",
			"threadId":     "t1",
			"labelIds":     []string{"INBOX"},
			"snippet":      "hello",
			"internalDate": "1700000000000",
			"payload": map[string]any{
				"mimeType": "multipart/mixed",
				"headers": []map[string]string{
					{"name": "From", "value": "Alice <alice@example.com>"},
					{"name": "Subject", "value": "Q3 roadmap"},
					{"name": "To", "value": "bob@example.com, Carol <carol@example.com>"},
				},
				"parts": []map[string]any{
					{
						"mimeType": "multipart/alternative",
						"parts": []map[string]any{
							{"mimeType": "text/plain", "body": map[string]any{"data": b64("plain body")}},
							{"mimeType": "text/html", "body": map[string]any{"data": b64("<p>html body</p>")}},
						},
					},
					{
						"mimeType": "application/pdf",
						"filename": "roadmap.pdf",
						"body":     map[string]any{"size": 1024, "attachmentId": "att-1"},
					},
				},
			},
		})
	}))

	m, err := p.Message(context.Background(), "m1")
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	if m.From != "Alice <alice@example.com>" || m.Subject != "Q3 roadmap" {
		t.Fatalf("headers wrong: %+v", m)
	}
	if len(m.To) != 2 || m.To[0] != "bob@example.com" || m.To[1] != "carol@example.com" {
		t.Fatalf("To = %v", m.To)
	}
	if m.BodyText != "plain body" || m.BodyHTML != "<p>html body</p>" {
		t.Fatalf("bodies = %q / %q", m.BodyText, m.BodyHTML)
	}
	if len(m.Attachments) != 1 || m.Attachments[0].Filename != "roadmap.pdf" || m.Attachments[0].Size != 1024 {
		t.Fatalf("Attachments = %+v", m.Attachments)
	}
	if len(m.Tags) != 1 || m.Tags[0] != "INBOX" {
		t.Fatalf("Tags = %v", m.Tags)
	}
	if m.Date.Unix() != 1700000000 {
		t.Fatalf("Date = %v", m.Date)
	}
}

func TestSearch(t *testing.T) {
	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/messages":
			if got := r.URL.Query().Get("q"); got != "roadmap" {
				t.Fatalf("q = %q, want roadmap", got)
			}
			writeJSON(t, w, map[string]any{"messages": []map[string]string{{"id": "m1"}}})
		case "/messages/m1":
			writeJSON(t, w, plainMessageFixture("m1", "Q3 roadmap"))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	msgs, err := p.Search(context.Background(), "roadmap")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Subject != "Q3 roadmap" {
		t.Fatalf("Search = %+v", msgs)
	}
}

func b64(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}

func plainMessageFixture(id, subject string) map[string]any {
	return map[string]any{
		"id":           id,
		"threadId":     "t-" + id,
		"internalDate": "1700000000000",
		"payload": map[string]any{
			"mimeType": "text/plain",
			"headers": []map[string]string{
				{"name": "Subject", "value": subject},
			},
			"body": map[string]any{"data": b64("body of " + id)},
		},
	}
}
