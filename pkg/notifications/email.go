// file: pkg/notifications/email.go
// version: 1.1.0
// guid: 7b26e0a9-4d31-4c58-91f7-2a05c3b8e6d4
// last-edited: 2026-07-26
package notifications

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPNotifier sends email using an SMTP server.
type SMTPNotifier struct {
	// Addr is the SMTP server address.
	Addr string
	// Auth provides optional authentication.
	Auth smtp.Auth
	// From is the sender address.
	From string
	// To holds destination email addresses.
	To []string
	// Send overrides the send function for testing.
	Send func(addr string, a smtp.Auth, from string, to []string, msg []byte) error
}

// Notify sends msg via SMTP.
func (s SMTPNotifier) Notify(ctx context.Context, msg string) error {
	send := s.Send
	if send == nil {
		send = smtp.SendMail
	}
	if s.Addr == "" || s.From == "" || len(s.To) == 0 {
		return fmt.Errorf("smtp configuration incomplete")
	}
	// Build the header block explicitly. Previously only a Subject was sent,
	// which many MTAs and spam filters reject outright for lacking From/To.
	//
	// msg is placed after the blank line that terminates the headers, so its
	// content cannot introduce one; a CR/LF inside it starts a new body line,
	// not a new header. Control characters are stripped from the header values
	// regardless, because those are the fields where injection would matter,
	// and Go's SMTP client dot-stuffs the body so a lone "." cannot end the
	// DATA command early.
	var b strings.Builder
	b.WriteString("From: " + sanitizeHeader(s.From) + "\r\n")
	b.WriteString("To: " + sanitizeHeader(strings.Join(s.To, ", ")) + "\r\n")
	b.WriteString("Subject: Subtitle Manager\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(msg)

	return send(s.Addr, s.Auth, s.From, s.To, []byte(b.String()))
}

// sanitizeHeader removes the characters that could terminate a header line and
// begin another, so an address taken from configuration cannot inject headers.
func sanitizeHeader(v string) string {
	return strings.Map(func(r rune) rune {
		if r == '\r' || r == '\n' {
			return -1
		}
		return r
	}, v)
}
