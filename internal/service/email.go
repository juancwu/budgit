package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/resend/resend-go/v2"
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

type EmailClient interface {
	SendWithContext(ctx context.Context, params *EmailParams) (string, error)
}

type ResendClient struct {
	client *resend.Client
}

func NewResendClient(apiKey string) *ResendClient {
	var client *resend.Client
	if apiKey != "" {
		client = resend.NewClient(apiKey)
	} else {
		slog.Warn("cannot initialize Resend client with empty api key")
		return nil
	}
	return &ResendClient{client: client}
}

func (c *ResendClient) SendWithContext(ctx context.Context, params *EmailParams) (string, error) {
	res, err := c.client.Emails.SendWithContext(ctx, &resend.SendEmailRequest{
		From:    params.From,
		To:      params.To,
		Bcc:     params.Bcc,
		Cc:      params.Cc,
		ReplyTo: params.ReplyTo,
		Subject: params.Subject,
		Text:    params.Text,
		Html:    params.Html,
	})
	if err != nil {
		return "", err
	}
	return res.Id, nil
}

type EmailService struct {
	client    EmailClient
	fromEmail string
	isDev     bool
	appURL    string
	appName   string
}

func NewEmailService(client EmailClient, fromEmail, appURL, appName string, isDev bool) *EmailService {
	return &EmailService{
		client:    client,
		fromEmail: fromEmail,
		isDev:     isDev,
		appURL:    appURL,
		appName:   appName,
	}
}

func (s *EmailService) SendMagicLinkEmail(email, token, name string) error {
	magicURL := fmt.Sprintf("%s/auth/magic-link/%s", s.appURL, token)
	subject, body := magicLinkEmailTemplate(magicURL, s.appName)

	if s.isDev {
		slog.Info("email sent (dev mode)", "type", "magic_link", "to", email, "subject", subject, "url", magicURL)
		return nil
	}

	if s.client == nil {
		return fmt.Errorf("email service not configured (missing RESEND_API_KEY)")
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
