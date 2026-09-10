package outlook

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

func TestSendPlain(t *testing.T) {
	var captured sendMailRequest
	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/sendMail" {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))

	err := p.Send(context.Background(), provider.Draft{
		To:           []string{"bob@example.com"},
		Cc:           []string{"carol@example.com"},
		Subject:      "Hello",
		MarkdownBody: "**hi** _there_",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if captured.Message.Subject != "Hello" {
		t.Fatalf("Subject = %q, want Hello", captured.Message.Subject)
	}
	if !captured.SaveToSentItems {
		t.Fatal("SaveToSentItems = false, want true")
	}
	if len(captured.Message.ToRecipients) != 1 || captured.Message.ToRecipients[0].EmailAddress.Address != "bob@example.com" {
		t.Fatalf("ToRecipients = %+v", captured.Message.ToRecipients)
	}
	if len(captured.Message.CcRecipients) != 1 || captured.Message.CcRecipients[0].EmailAddress.Address != "carol@example.com" {
		t.Fatalf("CcRecipients = %+v", captured.Message.CcRecipients)
	}
	if captured.Message.Body.ContentType != "HTML" {
		t.Fatalf("Body.ContentType = %q, want HTML", captured.Message.Body.ContentType)
	}
	if !strings.Contains(captured.Message.Body.Content, "<strong>hi</strong>") ||
		!strings.Contains(captured.Message.Body.Content, "<em>there</em>") {
		t.Fatalf("Body.Content = %q, want rendered markdown", captured.Message.Body.Content)
	}
	if len(captured.Message.Attachments) != 0 {
		t.Fatalf("Attachments = %+v, want none", captured.Message.Attachments)
	}
}

func TestSendWithAttachment(t *testing.T) {
	const attachmentBytes = "PDF-ish content"

	var captured sendMailRequest
	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))

	err := p.Send(context.Background(), provider.Draft{
		To:      []string{"bob@example.com"},
		Subject: "With attachment",
		Attachments: []provider.DraftAttachment{
			{Filename: "roadmap.pdf", MIMEType: "application/pdf", Content: strings.NewReader(attachmentBytes)},
		},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if len(captured.Message.Attachments) != 1 {
		t.Fatalf("Attachments = %+v, want 1", captured.Message.Attachments)
	}
	att := captured.Message.Attachments[0]
	if att.ODataType != "#microsoft.graph.fileAttachment" || att.Name != "roadmap.pdf" || att.ContentType != "application/pdf" {
		t.Fatalf("attachment metadata wrong: %+v", att)
	}
	decoded, err := base64.StdEncoding.DecodeString(att.ContentBytes)
	if err != nil {
		t.Fatalf("decode contentBytes: %v", err)
	}
	if string(decoded) != attachmentBytes {
		t.Fatalf("contentBytes decoded = %q, want %q", decoded, attachmentBytes)
	}
}

func TestSendErrorOnNonSuccessStatus(t *testing.T) {
	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))

	if err := p.Send(context.Background(), provider.Draft{Subject: "x"}); err == nil {
		t.Fatal("Send: expected error for non-2xx status, got nil")
	}
}
