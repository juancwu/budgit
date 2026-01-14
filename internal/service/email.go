package service

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/wneessen/go-mail"
)

type EmailParams struct {
	From    string
	To      []string
	Bcc     []string
	Cc      []string
	ReplyTo string
	Subject string
	Text    string
	Html    string
}

type EmailClient struct {
	smtpHost string
	smtpPort int
	imapHost string
	imapPort int
	username string
	password string
}

func NewEmailClient(smtpHost string, smtpPort int, imapHost string, imapPort int, username, password string) *EmailClient {
	return &EmailClient{
		smtpHost: smtpHost,
		smtpPort: smtpPort,
		imapHost: imapHost,
		imapPort: imapPort,
		username: username,
		password: password,
	}
}

func (nc *EmailClient) SendWithContext(ctx context.Context, params *EmailParams) (string, error) {
	m := mail.NewMsg()
	m.From(params.From)
	m.To(params.To...)
	m.Subject(params.Subject)

	if params.Html != "" {
		m.SetBodyString(mail.TypeTextHTML, params.Html)
		m.AddAlternativeString(mail.TypeTextPlain, params.Text)
	} else {
		m.SetBodyString(mail.TypeTextPlain, params.Text)
	}

	if params.ReplyTo != "" {
		m.ReplyTo(params.ReplyTo)
	}

	m.SetDate()
	m.SetMessageID()

	msgID := m.GetMessageID()

	var msgBuffer bytes.Buffer
	if _, err := m.WriteTo(&msgBuffer); err != nil {
		return "", fmt.Errorf("failed to buffer message: %w", err)
	}

	smtpClient, err := nc.connectToSMTP()
	if err != nil {
		return "", fmt.Errorf("failed to connect to SMTP server: %w", err)
	}

	err = smtpClient.DialAndSendWithContext(ctx, m)
	if err != nil {
		return "", fmt.Errorf("failed to send email: %w", err)
	}

	imapClient, err := nc.connectToIMAP()
	if err != nil {
		slog.Error("failed to establish connection with IMAP server", "error", err)
		return msgID, nil
	}
	defer imapClient.Logout()

	flags := []string{imap.SeenFlag}

	folderName := "Sent"

	literal := bytes.NewReader(msgBuffer.Bytes())

	err = imapClient.Append(folderName, flags, time.Now(), literal)
	if err != nil {
		slog.Error("IMAP append failed", "error", err)
	}

	return msgID, nil
}

func (nc *EmailClient) connectToSMTP() (*mail.Client, error) {
	smtpClient, err := mail.NewClient(
		nc.smtpHost,
		mail.WithPort(nc.smtpPort),
		mail.WithSMTPAuth(mail.SMTPAuthPlain),
		mail.WithUsername(nc.username),
		mail.WithPassword(nc.password),
		mail.WithTLSPolicy(mail.TLSMandatory),
	)
	return smtpClient, err
}

func (nc *EmailClient) connectToIMAP() (*client.Client, error) {
	var c *client.Client
	var err error

	addr := fmt.Sprintf("%s:%d", nc.imapHost, nc.imapPort)

	c, err = client.DialTLS(addr, nil)
	if err != nil {
		return nil, err
	}

	err = c.Login(nc.username, nc.password)
	if err != nil {
		return nil, err
	}

	return c, nil
}

type EmailService struct {
	client    *EmailClient
	fromEmail string
	isProd    bool
	appURL    string
	appName   string
}

func NewEmailService(client *EmailClient, fromEmail, appURL, appName string, isProd bool) *EmailService {
	return &EmailService{
		client:    client,
		fromEmail: fromEmail,
		isProd:    isProd,
		appURL:    appURL,
		appName:   appName,
	}
}

func (s *EmailService) SendMagicLinkEmail(email, token, name string) error {
	magicURL := fmt.Sprintf("%s/auth/magic-link/%s", s.appURL, token)
	subject, body := magicLinkEmailTemplate(magicURL, s.appName)

	if !s.isProd {
		slog.Info("email sent (dev mode)", "type", "magic_link", "to", email, "subject", subject, "url", magicURL)
		return nil
	}

	params := &EmailParams{
		From:    s.fromEmail,
		To:      []string{email},
		Subject: subject,
		Text:    body,
	}

	_, err := s.client.SendWithContext(context.Background(), params)
	if err == nil {
		slog.Info("email sent", "type", "magic_link", "to", email)
	}
	return err
}

func (s *EmailService) SendWelcomeEmail(email, name string) error {
	dashboardURL := fmt.Sprintf("%s/app/dashboard", s.appURL)
	subject, body := welcomeEmailTemplate(name, dashboardURL, s.appName)

	if !s.isProd {
		slog.Info("email sent (dev mode)", "type", "welcome", "to", email, "subject", subject, "url", dashboardURL)
		return nil
	}

	if s.client == nil {
		return fmt.Errorf("email service not configured")
	}

	params := &EmailParams{
		From:    s.fromEmail,
		To:      []string{email},
		Subject: subject,
		Text:    body,
	}

	_, err := s.client.SendWithContext(context.Background(), params)
	if err == nil {
		slog.Info("email sent", "type", "welcome", "to", email)
	}

	return err
}

func (s *EmailService) SendInvitationEmail(email, spaceName, inviterName, token string) error {
	inviteURL := fmt.Sprintf("%s/join/%s", s.appURL, token)
	subject, body := invitationEmailTemplate(spaceName, inviterName, inviteURL, s.appName)

	if !s.isProd {
		slog.Info("email sent (dev mode)", "type", "invitation", "to", email, "subject", subject, "url", inviteURL)
		return nil
	}

	params := &EmailParams{
		From:    s.fromEmail,
		To:      []string{email},
		Subject: subject,
		Text:    body,
	}

	_, err := s.client.SendWithContext(context.Background(), params)
	if err == nil {
		slog.Info("email sent", "type", "invitation", "to", email)
	}
	return err
}

func magicLinkEmailTemplate(magicURL, appName string) (string, string) {
	subject := fmt.Sprintf("Sign in to %s", appName)
	body := fmt.Sprintf(`Click this link to sign in to your account:
%s

This link expires in 10 minutes and can only be used once.

If you didn't request this, ignore this email.

Best,
The %s Team`, magicURL, appName)

	return subject, body
}

func welcomeEmailTemplate(name, dashboardURL, appName string) (string, string) {
	subject := fmt.Sprintf("Welcome to %s!", appName)
	body := fmt.Sprintf(`Hi %s,

Your email is verified and your account is active!

Get started: %s

If you have questions, reach out to our support team.

Best,
The %s Team`, name, dashboardURL, appName)

	return subject, body
}

func invitationEmailTemplate(spaceName, inviterName, inviteURL, appName string) (string, string) {
	subject := fmt.Sprintf("%s invited you to join %s on %s", inviterName, spaceName, appName)
	body := fmt.Sprintf(`Hi,

%s has invited you to join the space "%s" on %s.

Click the link below to accept the invitation:
%s

If you don't have an account, you will be asked to create one.

Best,
The %s Team`, inviterName, spaceName, appName, inviteURL, appName)

	return subject, body
}
