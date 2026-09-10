package outlook

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"testing"
)

func TestAttachment(t *testing.T) {
	const want = "the actual attachment bytes"

	p := newTestProvider(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/messages/m1/attachments/att-1" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		writeJSON(t, w, map[string]any{
			"contentBytes": base64.StdEncoding.EncodeToString([]byte(want)),
			"name":         "roadmap.pdf",
			"contentType":  "application/pdf",
		})
	}))

	rc, err := p.Attachment(context.Background(), "m1", "att-1")
	if err != nil {
		t.Fatalf("Attachment: %v", err)
	}
	defer func() {
		if err := rc.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	}()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read attachment: %v", err)
	}
	if string(got) != want {
		t.Fatalf("Attachment content = %q, want %q", got, want)
	}
}
