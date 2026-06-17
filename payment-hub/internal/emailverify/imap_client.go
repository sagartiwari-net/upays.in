package emailverify

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/emersion/go-message/mail"
)

type EmailMessage struct {
	MessageID string
	Subject   string
	Body      string
}

type IMAPClient struct {
	host, user, password, senderFilter string
	port                               int
}

func NewIMAPClient(host string, port int, user, password, senderFilter string) *IMAPClient {
	return &IMAPClient{
		host:         host,
		port:         port,
		user:         user,
		password:     password,
		senderFilter: strings.ToLower(senderFilter),
	}
}

func (c *IMAPClient) FetchRecent(ctx context.Context, since time.Time) ([]EmailMessage, error) {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	cli, err := client.DialTLS(addr, nil)
	if err != nil {
		return nil, fmt.Errorf("imap dial: %w", err)
	}
	defer cli.Logout() //nolint:errcheck

	if err := cli.Login(c.user, c.password); err != nil {
		return nil, fmt.Errorf("imap login: %w", err)
	}

	if _, err := cli.Select("INBOX", false); err != nil {
		return nil, fmt.Errorf("imap select inbox: %w", err)
	}

	criteria := imap.NewSearchCriteria()
	criteria.Since = since
	if c.senderFilter != "" {
		criteria.Header.Add("From", c.senderFilter)
	}

	uids, err := cli.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("imap search: %w", err)
	}
	if len(uids) == 0 {
		return nil, nil
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchBodyStructure, section.FetchItem(), imap.FetchUid}
	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- cli.Fetch(seqset, items, messages)
	}()

	var out []EmailMessage
	for msg := range messages {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if msg.Envelope == nil || len(msg.Envelope.From) == 0 {
			continue
		}
		from := strings.ToLower(msg.Envelope.From[0].Address())
		if c.senderFilter != "" && !strings.Contains(from, c.senderFilter) {
			continue
		}

		r := msg.GetBody(section)
		if r == nil {
			continue
		}

		body, err := readMailBody(r)
		if err != nil {
			continue
		}

		msgID := msg.Envelope.MessageId
		if msgID == "" {
			msgID = fmt.Sprintf("uid-%d", msg.Uid)
		}
		subject := ""
		if msg.Envelope != nil {
			subject = msg.Envelope.Subject
		}
		out = append(out, EmailMessage{MessageID: msgID, Subject: subject, Body: body})
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("imap fetch: %w", err)
	}
	return out, nil
}

func readMailBody(r io.Reader) (string, error) {
	mr, err := mail.CreateReader(r)
	if err != nil {
		b, readErr := io.ReadAll(r)
		if readErr != nil {
			return "", err
		}
		return string(b), nil
	}

	var parts []string
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		switch p.Header.(type) {
		case *mail.InlineHeader, *mail.AttachmentHeader:
			b, err := io.ReadAll(p.Body)
			if err != nil {
				continue
			}
			if len(b) > 0 {
				parts = append(parts, string(b))
			}
		}
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("empty mail body")
	}
	return strings.Join(parts, "\n"), nil
}

type IMAPTestResult struct {
	OK       bool
	Message  string
	Subjects []string
}

func (c *IMAPClient) TestConnection(ctx context.Context) (*IMAPTestResult, error) {
	addr := fmt.Sprintf("%s:%d", c.host, c.port)
	cli, err := client.DialTLS(addr, nil)
	if err != nil {
		return &IMAPTestResult{OK: false, Message: "IMAP connect failed: " + err.Error()}, nil
	}
	defer cli.Logout() //nolint:errcheck

	if err := cli.Login(c.user, c.password); err != nil {
		return &IMAPTestResult{OK: false, Message: "IMAP login failed: " + err.Error()}, nil
	}

	mbox, err := cli.Select("INBOX", false)
	if err != nil {
		return &IMAPTestResult{OK: false, Message: "INBOX select failed: " + err.Error()}, nil
	}

	criteria := imap.NewSearchCriteria()
	criteria.Since = time.Now().UTC().Add(-7 * 24 * time.Hour)
	if c.senderFilter != "" {
		criteria.Header.Add("From", c.senderFilter)
	}

	uids, err := cli.Search(criteria)
	if err != nil {
		return &IMAPTestResult{OK: false, Message: "search failed: " + err.Error()}, nil
	}

	subjects := []string{}
	if len(uids) > 0 {
		start := 0
		if len(uids) > 3 {
			start = len(uids) - 3
		}
		recent := uids[start:]
		seqset := new(imap.SeqSet)
		seqset.AddNum(recent...)
		messages := make(chan *imap.Message, 3)
		done := make(chan error, 1)
		go func() {
			done <- cli.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messages)
		}()
		for msg := range messages {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}
			if msg.Envelope != nil {
				subjects = append(subjects, msg.Envelope.Subject)
			}
		}
		if err := <-done; err != nil {
			return &IMAPTestResult{OK: false, Message: "fetch failed: " + err.Error()}, nil
		}
	}

	msg := fmt.Sprintf("Connected. INBOX has %d messages (last 7 days: %d)", mbox.Messages, len(uids))
	return &IMAPTestResult{OK: true, Message: msg, Subjects: subjects}, nil
}

func (c *IMAPClient) FetchLatestBody(ctx context.Context) (string, error) {
	since := time.Now().UTC().Add(-7 * 24 * time.Hour)
	messages, err := c.FetchRecent(ctx, since)
	if err != nil {
		return "", err
	}
	if len(messages) == 0 {
		return "", fmt.Errorf("no matching emails in last 7 days")
	}
	return messages[len(messages)-1].Body, nil
}
