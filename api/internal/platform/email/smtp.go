package email

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// smtpSender delivers mail through an SMTP relay.
type smtpSender struct {
	addr string
	auth smtp.Auth
	from string
}

// newSMTPSender validates the SMTP config and returns a Sender. Host and From
// are required; Port defaults to 587; PLAIN auth is used when a username is set.
func newSMTPSender(cfg SMTPConfig) (Sender, error) {
	if cfg.Host == "" || cfg.From == "" {
		return nil, fmt.Errorf("smtp email: SMTP_HOST and EMAIL_FROM are required")
	}
	port := cfg.Port
	if port == 0 {
		port = 587
	}
	var auth smtp.Auth
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	return &smtpSender{addr: fmt.Sprintf("%s:%d", cfg.Host, port), auth: auth, from: cfg.From}, nil
}

// Send delivers one plain-text email. The context is not honored by net/smtp;
// the caller already bounds the fan-out in a timed goroutine.
func (s *smtpSender) Send(_ context.Context, msg Message) error {
	return smtp.SendMail(s.addr, s.auth, s.from, []string{msg.To}, []byte(buildMIME(s.from, msg)))
}

// buildMIME assembles a minimal text/plain RFC 5322 message.
func buildMIME(from string, msg Message) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + msg.To + "\r\n")
	b.WriteString("Subject: " + msg.Subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=\"UTF-8\"\r\n\r\n")
	b.WriteString(msg.Body)
	return b.String()
}
