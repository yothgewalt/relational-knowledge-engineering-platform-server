package resend

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/resend/resend-go/v2"
)

type ResendConfig struct {
	ApiKey string
}

type HealthStatus struct {
	Connected bool          `json:"connected"`
	ApiKey    string        `json:"api_key"`
	Latency   time.Duration `json:"latency"`
	Error     string        `json:"error,omitempty"`
}

type EmailRequest struct {
	From        string               `json:"from"`
	To          []string             `json:"to"`
	Subject     string               `json:"subject"`
	Html        string               `json:"html,omitempty"`
	Text        string               `json:"text,omitempty"`
	Cc          []string             `json:"cc,omitempty"`
	Bcc         []string             `json:"bcc,omitempty"`
	ReplyTo     string               `json:"reply_to,omitempty"`
	Tags        []resend.Tag         `json:"tags,omitempty"`
	Headers     map[string]string    `json:"headers,omitempty"`
	Attachments []*resend.Attachment `json:"attachments,omitempty"`
}

type EmailResponse struct {
	ID string `json:"id"`
}

type ResendService interface {
	HealthCheck(ctx context.Context) HealthStatus
	SendEmail(ctx context.Context, request *EmailRequest) (*EmailResponse, error)
	SendBulkEmails(ctx context.Context, requests []*EmailRequest) ([]*EmailResponse, error)
	Close() error
}

type ResendClient struct {
	client *resend.Client
	config ResendConfig
	mu     sync.RWMutex
}

func NewClient(config ResendConfig) (*ResendClient, error) {
	if config.ApiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	client := resend.NewClient(config.ApiKey)

	return &ResendClient{
		client: client,
		config: config,
	}, nil
}

func (r *ResendClient) HealthCheck(ctx context.Context) HealthStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	start := time.Now()
	status := HealthStatus{
		ApiKey: r.maskApiKey(r.config.ApiKey),
	}

	testRequest := &resend.SendEmailRequest{
		From:    "test@example.com",
		To:      []string{"test@example.com"},
		Subject: "Health Check",
		Html:    "<p>This is a health check email (this should not be sent)</p>",
	}

	_, err := r.client.Emails.Send(testRequest)
	status.Latency = time.Since(start)

	if err != nil {
		if err.Error() == "The provided authorization grant is invalid, expired, revoked" {
			status.Connected = false
			status.Error = "Invalid API key"
		} else {
			status.Connected = true
			status.Error = ""
		}
	} else {
		status.Connected = true
		status.Error = ""
	}

	return status
}

func (r *ResendClient) SendEmail(ctx context.Context, request *EmailRequest) (*EmailResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if err := r.validateEmailRequest(request); err != nil {
		return nil, fmt.Errorf("invalid email request: %w", err)
	}

	resendRequest := &resend.SendEmailRequest{
		From:        request.From,
		To:          request.To,
		Subject:     request.Subject,
		Html:        request.Html,
		Text:        request.Text,
		Cc:          request.Cc,
		Bcc:         request.Bcc,
		ReplyTo:     request.ReplyTo,
		Tags:        request.Tags,
		Headers:     request.Headers,
		Attachments: request.Attachments,
	}

	sent, err := r.client.Emails.Send(resendRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to send email: %w", err)
	}

	return &EmailResponse{
		ID: sent.Id,
	}, nil
}

func (r *ResendClient) SendBulkEmails(ctx context.Context, requests []*EmailRequest) ([]*EmailResponse, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(requests) == 0 {
		return nil, fmt.Errorf("no email requests provided")
	}

	if len(requests) > 100 {
		return nil, fmt.Errorf("too many email requests, maximum 100 allowed")
	}

	var resendRequests []*resend.SendEmailRequest
	for _, request := range requests {
		if err := r.validateEmailRequest(request); err != nil {
			return nil, fmt.Errorf("invalid email request: %w", err)
		}

		resendRequests = append(resendRequests, &resend.SendEmailRequest{
			From:        request.From,
			To:          request.To,
			Subject:     request.Subject,
			Html:        request.Html,
			Text:        request.Text,
			Cc:          request.Cc,
			Bcc:         request.Bcc,
			ReplyTo:     request.ReplyTo,
			Tags:        request.Tags,
			Headers:     request.Headers,
			Attachments: request.Attachments,
		})
	}

	sent, err := r.client.Batch.Send(resendRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to send bulk emails: %w", err)
	}

	var responses []*EmailResponse
	for _, email := range sent.Data {
		responses = append(responses, &EmailResponse{
			ID: email.Id,
		})
	}

	return responses, nil
}

func (r *ResendClient) validateEmailRequest(request *EmailRequest) error {
	if request == nil {
		return fmt.Errorf("email request cannot be nil")
	}

	if request.From == "" {
		return fmt.Errorf("from address is required")
	}

	if len(request.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}

	if request.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if request.Html == "" && request.Text == "" {
		return fmt.Errorf("either HTML or text content is required")
	}

	for _, email := range request.To {
		if email == "" {
			return fmt.Errorf("empty email address in 'to' field")
		}
	}

	for _, email := range request.Cc {
		if email == "" {
			return fmt.Errorf("empty email address in 'cc' field")
		}
	}

	for _, email := range request.Bcc {
		if email == "" {
			return fmt.Errorf("empty email address in 'bcc' field")
		}
	}

	return nil
}

func (r *ResendClient) maskApiKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + "***" + apiKey[len(apiKey)-4:]
}

func (r *ResendClient) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.client = nil
	return nil
}
