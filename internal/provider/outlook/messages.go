package outlook

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

const pageSize = 50

const messageSelect = "id,subject,from,toRecipients,ccRecipients,receivedDateTime,bodyPreview,parentFolderId"

type emailAddress struct {
	Name    string `json:"name"`
	Address string `json:"address"`
}

type recipient struct {
	EmailAddress emailAddress `json:"emailAddress"`
}

type graphMessage struct {
	ID               string      `json:"id"`
	Subject          string      `json:"subject"`
	BodyPreview      string      `json:"bodyPreview"`
	ReceivedDateTime time.Time   `json:"receivedDateTime"`
	ParentFolderID   string      `json:"parentFolderId"`
	From             *recipient  `json:"from"`
	ToRecipients     []recipient `json:"toRecipients"`
	CcRecipients     []recipient `json:"ccRecipients"`
	Body             *struct {
		ContentType string `json:"contentType"`
		Content     string `json:"content"`
	} `json:"body"`
}

type messagesResponse struct {
	Value    []graphMessage `json:"value"`
	NextLink string         `json:"@odata.nextLink"`
}

// Messages lists a page of tag, newest first. Unlike Gmail, Graph's list
// endpoint returns the fields the UI needs directly via $select, so there's
// no per-message follow-up request.
func (p *Provider) Messages(ctx context.Context, tag provider.TagID, cursor provider.Cursor) (provider.Page, error) {
	path := string(cursor)
	if path == "" {
		q := url.Values{"$top": {fmt.Sprint(pageSize)}, "$select": {messageSelect}}
		path = "/mailFolders/" + url.PathEscape(string(tag)) + "/messages?" + q.Encode()
	}

	var resp messagesResponse
	if err := p.getJSON(ctx, path, &resp); err != nil {
		return provider.Page{}, err
	}

	return provider.Page{
		Messages: toMessages(resp.Value),
		Next:     provider.Cursor(resp.NextLink),
		HasMore:  resp.NextLink != "",
	}, nil
}

// Search runs a Graph $search query across all mail.
func (p *Provider) Search(ctx context.Context, query string) ([]provider.Message, error) {
	q := url.Values{"$top": {fmt.Sprint(pageSize)}, "$select": {messageSelect}, "$search": {`"` + query + `"`}}

	var resp messagesResponse
	if err := p.getJSON(ctx, "/messages?"+q.Encode(), &resp); err != nil {
		return nil, err
	}
	return toMessages(resp.Value), nil
}

func (p *Provider) Message(ctx context.Context, id provider.MessageID) (*provider.Message, error) {
	q := url.Values{"$select": {messageSelect + ",body"}}

	var gm graphMessage
	if err := p.getJSON(ctx, "/messages/"+url.PathEscape(string(id))+"?"+q.Encode(), &gm); err != nil {
		return nil, err
	}
	m := toMessage(gm)

	var atts attachmentsResponse
	if err := p.getJSON(ctx, "/messages/"+url.PathEscape(string(id))+"/attachments", &atts); err != nil {
		return nil, err
	}
	for _, a := range atts.Value {
		m.Attachments = append(m.Attachments, provider.Attachment{
			ID:       provider.AttachmentID(a.ID),
			Filename: a.Name,
			MIMEType: a.ContentType,
			Size:     int64(a.Size),
		})
	}

	return &m, nil
}

type attachmentsResponse struct {
	Value []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		ContentType string `json:"contentType"`
		Size        int    `json:"size"`
	} `json:"value"`
}

func toMessages(gms []graphMessage) []provider.Message {
	msgs := make([]provider.Message, len(gms))
	for i, gm := range gms {
		msgs[i] = toMessage(gm)
	}
	return msgs
}

func toMessage(gm graphMessage) provider.Message {
	m := provider.Message{
		ID:      provider.MessageID(gm.ID),
		Subject: gm.Subject,
		Snippet: gm.BodyPreview,
		Date:    gm.ReceivedDateTime,
		From:    formatAddress(gm.From),
		To:      formatAddresses(gm.ToRecipients),
		Cc:      formatAddresses(gm.CcRecipients),
	}
	if gm.ParentFolderID != "" {
		m.Tags = []provider.TagID{provider.TagID(gm.ParentFolderID)}
	}
	if gm.Body != nil {
		switch gm.Body.ContentType {
		case "html":
			m.BodyHTML = gm.Body.Content
		default:
			m.BodyText = gm.Body.Content
		}
	}
	return m
}

func formatAddress(r *recipient) string {
	if r == nil {
		return ""
	}
	if r.EmailAddress.Name == "" {
		return r.EmailAddress.Address
	}
	return fmt.Sprintf("%s <%s>", r.EmailAddress.Name, r.EmailAddress.Address)
}

func formatAddresses(rs []recipient) []string {
	if len(rs) == 0 {
		return nil
	}
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = formatAddress(&r)
	}
	return out
}
