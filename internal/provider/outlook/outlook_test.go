package outlook

import (
	"context"
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

func TestTagsFlattensNestedFolders(t *testing.T) {
	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mailFolders":
			writeJSON(t, w, map[string]any{"value": []map[string]any{
				{"id": "inbox-id", "displayName": "Inbox", "childFolderCount": 0},
				{"id": "work-id", "displayName": "Work", "childFolderCount": 1},
			}})
		case "/mailFolders/work-id/childFolders":
			writeJSON(t, w, map[string]any{"value": []map[string]any{
				{"id": "projects-id", "displayName": "Projects", "childFolderCount": 0},
			}})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	tags, err := p.Tags(context.Background())
	if err != nil {
		t.Fatalf("Tags: %v", err)
	}
	if len(tags) != 3 {
		t.Fatalf("Tags = %+v, want 3", tags)
	}
	if tags[0].Name != "Inbox" || tags[0].Special != provider.TagInbox {
		t.Fatalf("Tags[0] = %+v, want special Inbox", tags[0])
	}
	if tags[1].Name != "Work" || tags[1].Special != provider.TagNone {
		t.Fatalf("Tags[1] = %+v, want freeform Work", tags[1])
	}
	if tags[2].Name != "Work/Projects" {
		t.Fatalf("Tags[2].Name = %q, want Work/Projects", tags[2].Name)
	}
}

func TestMessagesPaginatesViaNextLink(t *testing.T) {
	var nextLinkURL string
	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/mailFolders/inbox-id/messages":
			writeJSON(t, w, map[string]any{
				"value":           []map[string]any{{"id": "m1", "subject": "First"}},
				"@odata.nextLink": nextLinkURL,
			})
		case "/page2":
			writeJSON(t, w, map[string]any{
				"value": []map[string]any{{"id": "m2", "subject": "Second"}},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	nextLinkURL = p.baseURL + "/page2"

	page1, err := p.Messages(context.Background(), "inbox-id", "")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if !page1.HasMore || string(page1.Next) != nextLinkURL {
		t.Fatalf("page1 = %+v, want HasMore + Next=%s", page1, nextLinkURL)
	}
	if len(page1.Messages) != 1 || page1.Messages[0].Subject != "First" {
		t.Fatalf("page1.Messages = %+v", page1.Messages)
	}

	page2, err := p.Messages(context.Background(), "inbox-id", page1.Next)
	if err != nil {
		t.Fatalf("Messages(page2): %v", err)
	}
	if page2.HasMore {
		t.Fatalf("page2.HasMore = true, want false")
	}
	if len(page2.Messages) != 1 || page2.Messages[0].Subject != "Second" {
		t.Fatalf("page2.Messages = %+v", page2.Messages)
	}
}

// TestMessagesIncludesBodyWithoutFollowUpFetch guards against a real bug:
// Messages (the list endpoint, used for initial sync) used to select only
// bodyPreview, not body, so every message synced on startup showed a blank
// preview until Watch happened to re-fetch it individually via Message.
// Graph's $select supports body on list endpoints too, so this must come
// back in the same call - no second request to this handler is registered,
// so a regression back to a follow-up fetch would 404 here.
func TestMessagesIncludesBodyWithoutFollowUpFetch(t *testing.T) {
	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mailFolders/inbox-id/messages" {
			t.Fatalf("unexpected path %s (body must come from the list call, not a follow-up fetch)", r.URL.Path)
		}
		writeJSON(t, w, map[string]any{"value": []map[string]any{
			{
				"id":      "m1",
				"subject": "Q3 roadmap",
				"body":    map[string]string{"contentType": "html", "content": "<p>hi</p>"},
			},
		}})
	}))

	page, err := p.Messages(context.Background(), "inbox-id", "")
	if err != nil {
		t.Fatalf("Messages: %v", err)
	}
	if len(page.Messages) != 1 || page.Messages[0].BodyHTML != "<p>hi</p>" {
		t.Fatalf("Messages = %+v, want BodyHTML populated from the list call", page.Messages)
	}
}

func TestMessageIncludesBodyAndAttachments(t *testing.T) {
	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/messages/m1":
			writeJSON(t, w, map[string]any{
				"id":               "m1",
				"subject":          "Q3 roadmap",
				"bodyPreview":      "hello",
				"receivedDateTime": "2023-11-14T22:13:20Z",
				"parentFolderId":   "inbox-id",
				"from":             map[string]any{"emailAddress": map[string]string{"name": "Alice", "address": "alice@example.com"}},
				"toRecipients": []map[string]any{
					{"emailAddress": map[string]string{"name": "", "address": "bob@example.com"}},
				},
				"body": map[string]string{"contentType": "html", "content": "<p>hi</p>"},
			})
		case "/messages/m1/attachments":
			writeJSON(t, w, map[string]any{"value": []map[string]any{
				{"id": "att-1", "name": "roadmap.pdf", "contentType": "application/pdf", "size": 1024},
			}})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))

	m, err := p.Message(context.Background(), "m1")
	if err != nil {
		t.Fatalf("Message: %v", err)
	}
	if m.From != "Alice <alice@example.com>" || m.Subject != "Q3 roadmap" {
		t.Fatalf("headers wrong: %+v", m)
	}
	if len(m.To) != 1 || m.To[0] != "bob@example.com" {
		t.Fatalf("To = %v", m.To)
	}
	if m.BodyHTML != "<p>hi</p>" {
		t.Fatalf("BodyHTML = %q", m.BodyHTML)
	}
	if len(m.Tags) != 1 || m.Tags[0] != "inbox-id" {
		t.Fatalf("Tags = %v", m.Tags)
	}
	if len(m.Attachments) != 1 || m.Attachments[0].Filename != "roadmap.pdf" || m.Attachments[0].Size != 1024 {
		t.Fatalf("Attachments = %+v", m.Attachments)
	}
}

func TestSearch(t *testing.T) {
	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("$search"); got != `"roadmap"` {
			t.Fatalf("$search = %q, want \"roadmap\"", got)
		}
		writeJSON(t, w, map[string]any{"value": []map[string]any{{"id": "m1", "subject": "Q3 roadmap"}}})
	}))

	msgs, err := p.Search(context.Background(), "roadmap")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(msgs) != 1 || msgs[0].Subject != "Q3 roadmap" {
		t.Fatalf("Search = %+v", msgs)
	}
}
