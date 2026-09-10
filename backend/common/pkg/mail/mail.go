// Package mail defines the outbound email interface and simple implementations.
package mail

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
)

// Message is a plain-text email.
type Message struct {
	To      string
	Subject string
	Body    string
}

// Sender delivers messages.
type Sender interface {
	Send(ctx context.Context, m Message) error
}

// Config configures outbound email. Provider "log" prints to stdout.
type Config struct {
	Provider string // log | smtp
	From     string
	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
}

// New picks an implementation from config.
func New(cfg Config) Sender {
	if cfg.Provider == "smtp" && cfg.SMTPHost != "" {
		return &smtpSender{cfg: cfg}
	}
	return &logSender{from: cfg.From}
}

// Discard returns a sender that drops every message (admin portal, which
// hands links out of band instead of emailing).
func Discard() Sender { return discardSender{} }

type discardSender struct{}

func (discardSender) Send(context.Context, Message) error { return nil }

type logSender struct{ from string }

func (l *logSender) Send(_ context.Context, m Message) error {
	slog.Info("mail (log provider)", "from", l.from, "to", m.To, "subject", m.Subject, "body", m.Body)
	return nil
}

type smtpSender struct{ cfg Config }

func (s *smtpSender) Send(_ context.Context, m Message) error {
	addr := fmt.Sprintf("%s:%d", s.cfg.SMTPHost, s.cfg.SMTPPort)
	var auth smtp.Auth
	if s.cfg.SMTPUser != "" {
		auth = smtp.PlainAuth("", s.cfg.SMTPUser, s.cfg.SMTPPass, s.cfg.SMTPHost)
	}
	msg := strings.Join([]string{
		"From: " + s.cfg.From,
		"To: " + m.To,
		"Subject: " + m.Subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"",
		m.Body,
	}, "\r\n")
	return smtp.SendMail(addr, auth, s.cfg.From, []string{m.To}, []byte(msg))
}
