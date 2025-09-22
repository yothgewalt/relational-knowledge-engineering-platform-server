package resend

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/resend/resend-go/v2"
)

func TestResendConfig_Validation(t *testing.T) {
	tests := []struct {
		name   string
		config ResendConfig
		valid  bool
	}{
		{
			name: "valid config",
			config: ResendConfig{
				ApiKey: "re_1234567890abcdef",
			},
			valid: true,
		},
		{
			name: "empty api key",
			config: ResendConfig{
				ApiKey: "",
			},
			valid: false,
		},
		{
			name: "short api key",
			config: ResendConfig{
				ApiKey: "re_123",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewResendClient(tt.config)
			if tt.valid && err != nil {
				t.Errorf("expected valid config but got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid config but got no error")
			}
		})
	}
}

func TestHealthStatus_Structure(t *testing.T) {
	status := HealthStatus{
		Connected: true,
		ApiKey:    "re_1***cdef",
		Latency:   25 * time.Millisecond,
		Error:     "",
	}

	if !status.Connected {
		t.Error("Connected field should be accessible")
	}
	if status.ApiKey != "re_1***cdef" {
		t.Error("ApiKey field should be accessible")
	}
	if status.Latency != 25*time.Millisecond {
		t.Error("Latency field should be accessible")
	}
	if status.Error != "" {
		t.Error("Error field should be empty")
	}
}

func TestResendService_Interface(t *testing.T) {
	var _ ResendService = (*ResendClient)(nil)
}

func TestResendClient_StructureAndMethods(t *testing.T) {
	config := ResendConfig{
		ApiKey: "re_test_key_1234567890",
	}

	client, err := NewResendClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	if client.config.ApiKey != "re_test_key_1234567890" {
		t.Error("config.ApiKey should be accessible")
	}

	if client.client == nil {
		t.Error("client should not be nil")
	}
}

func TestEmailRequest_Structure(t *testing.T) {
	request := &EmailRequest{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Email",
		Html:    "<h1>Hello World</h1>",
		Text:    "Hello World",
		Cc:      []string{"cc@example.com"},
		Bcc:     []string{"bcc@example.com"},
		ReplyTo: "replyto@example.com",
		Tags: []resend.Tag{
			{Name: "category", Value: "test"},
		},
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
		},
	}

	if request.From != "sender@example.com" {
		t.Error("From field should be accessible")
	}
	if len(request.To) != 1 || request.To[0] != "recipient@example.com" {
		t.Error("To field should be accessible")
	}
	if request.Subject != "Test Email" {
		t.Error("Subject field should be accessible")
	}
	if request.Html != "<h1>Hello World</h1>" {
		t.Error("Html field should be accessible")
	}
	if request.Text != "Hello World" {
		t.Error("Text field should be accessible")
	}
	if len(request.Cc) != 1 || request.Cc[0] != "cc@example.com" {
		t.Error("Cc field should be accessible")
	}
	if len(request.Bcc) != 1 || request.Bcc[0] != "bcc@example.com" {
		t.Error("Bcc field should be accessible")
	}
	if request.ReplyTo != "replyto@example.com" {
		t.Error("ReplyTo field should be accessible")
	}
	if len(request.Tags) != 1 || request.Tags[0].Name != "category" {
		t.Error("Tags field should be accessible")
	}
	if request.Headers["X-Custom-Header"] != "test-value" {
		t.Error("Headers field should be accessible")
	}
}

func TestEmailResponse_Structure(t *testing.T) {
	response := &EmailResponse{
		ID: "email-id-12345",
	}

	if response.ID != "email-id-12345" {
		t.Error("ID field should be accessible")
	}
}

func TestEmailRequest_Validation(t *testing.T) {
	config := ResendConfig{
		ApiKey: "re_test_key_1234567890",
	}

	client, err := NewResendClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	tests := []struct {
		name    string
		request *EmailRequest
		valid   bool
	}{
		{
			name: "valid email request",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Email",
				Html:    "<h1>Hello World</h1>",
			},
			valid: true,
		},
		{
			name:    "nil request",
			request: nil,
			valid:   false,
		},
		{
			name: "empty from address",
			request: &EmailRequest{
				From:    "",
				To:      []string{"recipient@example.com"},
				Subject: "Test Email",
				Html:    "<h1>Hello World</h1>",
			},
			valid: false,
		},
		{
			name: "empty to addresses",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{},
				Subject: "Test Email",
				Html:    "<h1>Hello World</h1>",
			},
			valid: false,
		},
		{
			name: "empty subject",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "",
				Html:    "<h1>Hello World</h1>",
			},
			valid: false,
		},
		{
			name: "no content",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Email",
				Html:    "",
				Text:    "",
			},
			valid: false,
		},
		{
			name: "empty email in to field",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com", ""},
				Subject: "Test Email",
				Html:    "<h1>Hello World</h1>",
			},
			valid: false,
		},
		{
			name: "empty email in cc field",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Cc:      []string{""},
				Subject: "Test Email",
				Html:    "<h1>Hello World</h1>",
			},
			valid: false,
		},
		{
			name: "empty email in bcc field",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Bcc:     []string{""},
				Subject: "Test Email",
				Html:    "<h1>Hello World</h1>",
			},
			valid: false,
		},
		{
			name: "text only content",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Email",
				Text:    "Hello World",
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.validateEmailRequest(tt.request)
			if tt.valid && err != nil {
				t.Errorf("expected valid request but got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid request but got no error")
			}
		})
	}
}

func TestBulkEmail_Validation(t *testing.T) {
	config := ResendConfig{
		ApiKey: "re_test_key_1234567890",
	}

	client, err := NewResendClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	tests := []struct {
		name     string
		requests []*EmailRequest
		valid    bool
	}{
		{
			name:     "empty requests",
			requests: []*EmailRequest{},
			valid:    false,
		},
		{
			name: "single valid request",
			requests: []*EmailRequest{
				{
					From:    "sender@example.com",
					To:      []string{"recipient@example.com"},
					Subject: "Test Email",
					Html:    "<h1>Hello World</h1>",
				},
			},
			valid: true,
		},
		{
			name: "multiple valid requests",
			requests: []*EmailRequest{
				{
					From:    "sender@example.com",
					To:      []string{"recipient1@example.com"},
					Subject: "Test Email 1",
					Html:    "<h1>Hello World 1</h1>",
				},
				{
					From:    "sender@example.com",
					To:      []string{"recipient2@example.com"},
					Subject: "Test Email 2",
					Html:    "<h1>Hello World 2</h1>",
				},
			},
			valid: true,
		},
		{
			name: "too many requests",
			requests: func() []*EmailRequest {
				requests := make([]*EmailRequest, 101)
				for i := range requests {
					requests[i] = &EmailRequest{
						From:    "sender@example.com",
						To:      []string{"recipient@example.com"},
						Subject: "Test Email",
						Html:    "<h1>Hello World</h1>",
					}
				}
				return requests
			}(),
			valid: false,
		},
		{
			name: "contains invalid request",
			requests: []*EmailRequest{
				{
					From:    "sender@example.com",
					To:      []string{"recipient@example.com"},
					Subject: "Test Email",
					Html:    "<h1>Hello World</h1>",
				},
				{
					From:    "",
					To:      []string{"recipient@example.com"},
					Subject: "Test Email",
					Html:    "<h1>Hello World</h1>",
				},
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			_, err := client.SendBulkEmails(ctx, tt.requests)

			if tt.valid {
				if err != nil && 
					!strings.Contains(err.Error(), "API key is invalid") && 
					!strings.Contains(err.Error(), "The provided authorization grant is invalid, expired, revoked") {
					t.Errorf("expected valid bulk request but got unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("expected invalid bulk request but got no error")
				}
			}
		})
	}
}

func TestApiKey_Masking(t *testing.T) {
	config := ResendConfig{
		ApiKey: "re_test_key_1234567890",
	}

	client, err := NewResendClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "normal key",
			apiKey:   "re_test_key_1234567890",
			expected: "re_t***7890",
		},
		{
			name:     "short key",
			apiKey:   "re_123",
			expected: "***",
		},
		{
			name:     "very short key",
			apiKey:   "123",
			expected: "***",
		},
		{
			name:     "exactly 8 chars",
			apiKey:   "12345678",
			expected: "***",
		},
		{
			name:     "9 chars",
			apiKey:   "123456789",
			expected: "1234***6789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			masked := client.maskApiKey(tt.apiKey)
			if masked != tt.expected {
				t.Errorf("expected masked key %s, got %s", tt.expected, masked)
			}
		})
	}
}

func TestResendClient_Close(t *testing.T) {
	config := ResendConfig{
		ApiKey: "re_test_key_1234567890",
	}

	client, err := NewResendClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	err = client.Close()
	if err != nil {
		t.Errorf("Close should not return error, got: %v", err)
	}

	if client.client != nil {
		t.Error("client should be nil after Close")
	}
}

func TestContext_Handling(t *testing.T) {
	config := ResendConfig{
		ApiKey: "re_test_key_1234567890",
	}

	_, err := NewResendClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	tests := []struct {
		name  string
		ctx   context.Context
		valid bool
	}{
		{
			name:  "valid context",
			ctx:   context.Background(),
			valid: true,
		},
		{
			name: "context with timeout",
			ctx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
				_ = cancel
				return ctx
			}(),
			valid: true,
		},
		{
			name:  "cancelled context",
			ctx:   func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx }(),
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ctx.Err()
			if tt.valid && err != nil {
				t.Errorf("expected valid context but got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid context but got no error")
			}
		})
	}
}

func TestConcurrentAccess(t *testing.T) {
	config := ResendConfig{
		ApiKey: "re_test_key_1234567890",
	}

	client, err := NewResendClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	done := make(chan bool, 5)

	for range 5 {
		go func() {
			defer func() { done <- true }()

			_ = client.config.ApiKey
			_ = client.maskApiKey(client.config.ApiKey)

			ctx := context.Background()
			_ = client.HealthCheck(ctx)
		}()
	}

	for range 5 {
		<-done
	}
}

func TestHealthCheck_Structure(t *testing.T) {
	config := ResendConfig{
		ApiKey: "re_test_key_1234567890",
	}

	client, err := NewResendClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	ctx := context.Background()
	health := client.HealthCheck(ctx)

	if health.ApiKey == "" {
		t.Error("health check should include masked API key")
	}

	if health.Latency == 0 {
		t.Error("health check should measure latency")
	}

	expectedMasked := client.maskApiKey(config.ApiKey)
	if health.ApiKey != expectedMasked {
		t.Errorf("expected masked API key %s, got %s", expectedMasked, health.ApiKey)
	}
}

func TestEmailRequest_EdgeCases(t *testing.T) {
	config := ResendConfig{
		ApiKey: "re_test_key_1234567890",
	}

	client, err := NewResendClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}

	tests := []struct {
		name    string
		request *EmailRequest
		valid   bool
	}{
		{
			name: "multiple recipients",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient1@example.com", "recipient2@example.com", "recipient3@example.com"},
				Subject: "Test Email",
				Html:    "<h1>Hello World</h1>",
			},
			valid: true,
		},
		{
			name: "both html and text",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Email",
				Html:    "<h1>Hello World</h1>",
				Text:    "Hello World",
			},
			valid: true,
		},
		{
			name: "with attachments",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Email",
				Html:    "<h1>Hello World</h1>",
				Attachments: []*resend.Attachment{
					{
						Filename: "test.txt",
						Content:  []byte("test content"),
					},
				},
			},
			valid: true,
		},
		{
			name: "with custom headers",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Email",
				Html:    "<h1>Hello World</h1>",
				Headers: map[string]string{
					"X-Priority":     "1",
					"X-Custom-Field": "custom-value",
				},
			},
			valid: true,
		},
		{
			name: "with tags",
			request: &EmailRequest{
				From:    "sender@example.com",
				To:      []string{"recipient@example.com"},
				Subject: "Test Email",
				Html:    "<h1>Hello World</h1>",
				Tags: []resend.Tag{
					{Name: "category", Value: "newsletter"},
					{Name: "priority", Value: "high"},
				},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.validateEmailRequest(tt.request)
			if tt.valid && err != nil {
				t.Errorf("expected valid request but got error: %v", err)
			}
			if !tt.valid && err == nil {
				t.Error("expected invalid request but got no error")
			}
		})
	}
}