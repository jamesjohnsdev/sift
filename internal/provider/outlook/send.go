package outlook

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

type sendMailRequest struct {
	Message         graphDraftMessage `json:"message"`
	SaveToSentItems bool              `json:"saveToSentItems"`
}

type graphDraftMessage struct {
	Subject       string            `json:"subject"`
	Body          graphItemBody     `json:"body"`
	ToRecipients  []recipient       `json:"toRecipients,omitempty"`
	CcRecipients  []recipient       `json:"ccRecipients,omitempty"`
	BccRecipients []recipient       `json:"bccRecipients,omitempty"`
	Attachments   []graphFileAttach `json:"attachments,omitempty"`
}

type graphItemBody struct {
	ContentType string `json:"contentType"`
	Content     string `json:"content"`
}

type graphFileAttach struct {
	ODataType    string `json:"@odata.type"`
	Name         string `json:"name"`
	ContentType  string `json:"contentType"`
	ContentBytes string `json:"contentBytes"`
}

// Send renders draft.MarkdownBody to HTML and sends it via Graph's
// sendMail action.
func (p *Provider) Send(ctx context.Context, draft provider.Draft) error {
	html, err := provider.RenderMarkdown(draft.MarkdownBody)
	if err != nil {
		return err
	}

	msg := graphDraftMessage{
		Subject:       draft.Subject,
		Body:          graphItemBody{ContentType: "HTML", Content: html},
		ToRecipients:  addressRecipients(draft.To),
		CcRecipients:  addressRecipients(draft.Cc),
		BccRecipients: addressRecipients(draft.Bcc),
	}

	for _, a := range draft.Attachments {
		content, err := io.ReadAll(a.Content)
		if err != nil {
			return fmt.Errorf("read attachment %s: %w", a.Filename, err)
		}
		msg.Attachments = append(msg.Attachments, graphFileAttach{
			ODataType:    "#microsoft.graph.fileAttachment",
			Name:         a.Filename,
			ContentType:  a.MIMEType,
			ContentBytes: base64.StdEncoding.EncodeToString(content),
		})
	}

	return p.postJSON(ctx, "/sendMail", sendMailRequest{Message: msg, SaveToSentItems: true})
}

func addressRecipients(addrs []string) []recipient {
	if len(addrs) == 0 {
		return nil
	}
	out := make([]recipient, len(addrs))
	for i, addr := range addrs {
		out[i] = recipient{EmailAddress: emailAddress{Address: addr}}
	}
	return out
}
