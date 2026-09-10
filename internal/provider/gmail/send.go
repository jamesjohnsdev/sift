package gmail

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

func (p *Provider) Send(ctx context.Context, draft provider.Draft) error {
	raw, err := buildRawMessage(draft)
	if err != nil {
		return fmt.Errorf("build message: %w", err)
	}

	body := map[string]string{"raw": base64.RawURLEncoding.EncodeToString(raw)}
	return p.postJSON(ctx, "/messages/send", body, nil)
}

// buildRawMessage assembles an RFC 2822 message: a multipart/alternative
// plaintext-Markdown + rendered-HTML body, wrapped in multipart/mixed with
// one part per attachment when there are any. It's sent to Gmail whole,
// base64url-encoded, via the messages.send `raw` field.
func buildRawMessage(draft provider.Draft) ([]byte, error) {
	html, err := provider.RenderMarkdown(draft.MarkdownBody)
	if err != nil {
		return nil, err
	}

	altBody, altBoundary, err := buildAlternativeBody(draft.MarkdownBody, html)
	if err != nil {
		return nil, fmt.Errorf("build alternative body: %w", err)
	}

	var msg bytes.Buffer
	headers := []struct{ key, value string }{
		{"MIME-Version", "1.0"},
		{"To", strings.Join(draft.To, ", ")},
		{"Cc", strings.Join(draft.Cc, ", ")},
		{"Bcc", strings.Join(draft.Bcc, ", ")},
		{"Subject", draft.Subject},
	}
	for _, h := range headers {
		if err := writeHeader(&msg, h.key, h.value); err != nil {
			return nil, err
		}
	}

	if len(draft.Attachments) == 0 {
		fmt.Fprintf(&msg, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", altBoundary)
		msg.Write(altBody)
		return msg.Bytes(), nil
	}

	mixedBody, mixedBoundary, err := buildMixedBody(altBody, altBoundary, draft.Attachments)
	if err != nil {
		return nil, fmt.Errorf("build mixed body: %w", err)
	}
	fmt.Fprintf(&msg, "Content-Type: multipart/mixed; boundary=%q\r\n\r\n", mixedBoundary)
	msg.Write(mixedBody)
	return msg.Bytes(), nil
}

func writeHeader(w io.Writer, key, value string) error {
	if value == "" {
		return nil
	}
	_, err := fmt.Fprintf(w, "%s: %s\r\n", key, value)
	return err
}

// buildAlternativeBody returns a finished multipart/alternative part
// (plaintext Markdown source + rendered HTML) and the boundary it used.
func buildAlternativeBody(markdown, html string) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	plain, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/plain; charset=UTF-8"},
		"Content-Transfer-Encoding": {"8bit"},
	})
	if err != nil {
		return nil, "", err
	}
	if _, err := io.WriteString(plain, markdown); err != nil {
		return nil, "", err
	}

	htmlPart, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Type":              {"text/html; charset=UTF-8"},
		"Content-Transfer-Encoding": {"8bit"},
	})
	if err != nil {
		return nil, "", err
	}
	if _, err := io.WriteString(htmlPart, html); err != nil {
		return nil, "", err
	}

	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), w.Boundary(), nil
}

// buildMixedBody wraps a finished multipart/alternative body as the first
// part of a multipart/mixed message, followed by one base64 part per
// attachment.
func buildMixedBody(altBody []byte, altBoundary string, attachments []provider.DraftAttachment) ([]byte, string, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	altPart, err := w.CreatePart(textproto.MIMEHeader{
		"Content-Type": {fmt.Sprintf("multipart/alternative; boundary=%q", altBoundary)},
	})
	if err != nil {
		return nil, "", err
	}
	if _, err := altPart.Write(altBody); err != nil {
		return nil, "", err
	}

	for _, a := range attachments {
		part, err := w.CreatePart(textproto.MIMEHeader{
			"Content-Type":              {a.MIMEType},
			"Content-Transfer-Encoding": {"base64"},
			"Content-Disposition":       {fmt.Sprintf("attachment; filename=%q", a.Filename)},
		})
		if err != nil {
			return nil, "", err
		}
		enc := base64.NewEncoder(base64.StdEncoding, part)
		if _, err := io.Copy(enc, a.Content); err != nil {
			return nil, "", err
		}
		if err := enc.Close(); err != nil {
			return nil, "", err
		}
	}

	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), w.Boundary(), nil
}
