package gmail

import (
	"context"
	"encoding/base64"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jamesjohnsdev/sift/internal/provider"
)

const pageSize = 50

type messageRef struct {
	ID string `json:"id"`
}

type messageListResponse struct {
	Messages      []messageRef `json:"messages"`
	NextPageToken string       `json:"nextPageToken"`
}

// Messages lists a page of tag, newest first. Gmail's list endpoint only
// returns message IDs, so each one is then fetched (concurrently, bounded)
// for the headers/snippet the UI needs.
func (p *Provider) Messages(ctx context.Context, tag provider.TagID, cursor provider.Cursor) (provider.Page, error) {
	q := url.Values{"maxResults": {strconv.Itoa(pageSize)}, "labelIds": {string(tag)}}
	if cursor != "" {
		q.Set("pageToken", string(cursor))
	}
	return p.listAndFetch(ctx, "/messages?"+q.Encode())
}

// Search runs a Gmail search-operator query across all mail.
func (p *Provider) Search(ctx context.Context, query string) ([]provider.Message, error) {
	q := url.Values{"maxResults": {strconv.Itoa(pageSize)}, "q": {query}}
	page, err := p.listAndFetch(ctx, "/messages?"+q.Encode())
	if err != nil {
		return nil, err
	}
	return page.Messages, nil
}

func (p *Provider) listAndFetch(ctx context.Context, path string) (provider.Page, error) {
	var list messageListResponse
	if err := p.getJSON(ctx, path, &list); err != nil {
		return provider.Page{}, err
	}

	msgs, err := p.fetchMessages(ctx, list.Messages)
	if err != nil {
		return provider.Page{}, err
	}

	return provider.Page{
		Messages: msgs,
		Next:     provider.Cursor(list.NextPageToken),
		HasMore:  list.NextPageToken != "",
	}, nil
}

// fetchMessages fetches each message's full detail concurrently (Gmail has
// no equivalent to Graph's inline $select on a list response), bounded to
// avoid hammering the API on a large page.
func (p *Provider) fetchMessages(ctx context.Context, refs []messageRef) ([]provider.Message, error) {
	msgs := make([]provider.Message, len(refs))
	errs := make([]error, len(refs))

	const workers = 10
	sem := make(chan struct{}, workers)
	var wg sync.WaitGroup
	for i, ref := range refs {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, id string) {
			defer wg.Done()
			defer func() { <-sem }()
			m, err := p.Message(ctx, provider.MessageID(id))
			if err != nil {
				errs[i] = err
				return
			}
			msgs[i] = *m
		}(i, ref.ID)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	return msgs, nil
}

type gmailPart struct {
	MimeType string      `json:"mimeType"`
	Filename string      `json:"filename"`
	Headers  []gmailHdr  `json:"headers"`
	Body     gmailBody   `json:"body"`
	Parts    []gmailPart `json:"parts"`
}

type gmailHdr struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type gmailBody struct {
	Size         int    `json:"size"`
	Data         string `json:"data"`
	AttachmentID string `json:"attachmentId"`
}

type messageResponse struct {
	ID           string    `json:"id"`
	ThreadID     string    `json:"threadId"`
	LabelIDs     []string  `json:"labelIds"`
	Snippet      string    `json:"snippet"`
	InternalDate string    `json:"internalDate"`
	Payload      gmailPart `json:"payload"`
}

func (p *Provider) Message(ctx context.Context, id provider.MessageID) (*provider.Message, error) {
	var resp messageResponse
	if err := p.getJSON(ctx, "/messages/"+url.PathEscape(string(id))+"?format=full", &resp); err != nil {
		return nil, err
	}

	m := &provider.Message{
		ID:       id,
		ThreadID: resp.ThreadID,
		Snippet:  resp.Snippet,
		From:     header(resp.Payload.Headers, "From"),
		Subject:  header(resp.Payload.Headers, "Subject"),
		To:       addressList(header(resp.Payload.Headers, "To")),
		Cc:       addressList(header(resp.Payload.Headers, "Cc")),
	}
	for _, l := range resp.LabelIDs {
		m.Tags = append(m.Tags, provider.TagID(l))
	}
	if ms, err := strconv.ParseInt(resp.InternalDate, 10, 64); err == nil {
		m.Date = time.UnixMilli(ms)
	}

	walkParts(resp.Payload, m)
	return m, nil
}

// walkParts recurses the MIME tree, collecting the first text/plain and
// text/html bodies plus any part carrying a filename as an attachment.
func walkParts(part gmailPart, m *provider.Message) {
	switch {
	case part.Filename != "":
		size := part.Body.Size
		m.Attachments = append(m.Attachments, provider.Attachment{
			ID:       provider.AttachmentID(part.Body.AttachmentID),
			Filename: part.Filename,
			MIMEType: part.MimeType,
			Size:     int64(size),
		})
	case part.MimeType == "text/plain" && m.BodyText == "":
		m.BodyText = decodeBody(part.Body.Data)
	case part.MimeType == "text/html" && m.BodyHTML == "":
		m.BodyHTML = decodeBody(part.Body.Data)
	}
	for _, child := range part.Parts {
		walkParts(child, m)
	}
}

func decodeBody(data string) string {
	if data == "" {
		return ""
	}
	b, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		return ""
	}
	return string(b)
}

func header(headers []gmailHdr, name string) string {
	for _, h := range headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

func addressList(raw string) []string {
	if raw == "" {
		return nil
	}
	addrs, err := mail.ParseAddressList(raw)
	if err != nil {
		return []string{raw}
	}
	out := make([]string, len(addrs))
	for i, a := range addrs {
		out[i] = a.Address
	}
	return out
}
