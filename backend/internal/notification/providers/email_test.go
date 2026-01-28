package providers

import (
	"logflux/model"
	"testing"
)

func TestEmailProvider_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"smtp_host": "smtp.example.com",
				"smtp_port": 587,
				"username":  "user",
				"password":  "pass",
				"from":      "user@example.com",
				"to":        []string{"admin@example.com"},
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: map[string]interface{}{
				"smtp_port": 587,
				"username":  "user",
				"password":  "pass",
				"from":      "user@example.com",
				"to":        []string{"admin@example.com"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &EmailProvider{}
			if err := p.Validate(tt.config); (err != nil) != tt.wantErr {
				t.Errorf("EmailProvider.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEmailProvider_Type(t *testing.T) {
	p := &EmailProvider{}
	if got := p.Type(); got != model.ChannelTypeEmail {
		t.Errorf("EmailProvider.Type() = %v, want %v", got, model.ChannelTypeEmail)
	}
}

func TestNewEmailProvider(t *testing.T) {
	// config := map[string]interface{}{
	// 	"smtp_host": "smtp.example.com",
	// 	"smtp_port": 587,
	// 	"username":  "user",
	// 	"password":  "pass",
	// 	"from":      "user@example.com",
	// 	"to":        []string{"admin@example.com"},
	// }

	p := NewEmailProvider()
	if p == nil {
		t.Error("NewEmailProvider() returned nil")
	}

	// Check internal structure if needed, or just type
	if p.Type() != model.ChannelTypeEmail {
		t.Errorf("Type() = %v, want %v", p.Type(), model.ChannelTypeEmail)
	}
}
