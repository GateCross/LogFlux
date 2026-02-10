package providers

import (
	"logflux/model"
	"testing"
)

func TestWeChatMPProvider_Validate(t *testing.T) {
	provider := NewWeChatMPProvider()

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"app_id":     "wx1234567890",
				"app_secret": "secret",
				"to_user":    "openid-xxx",
			},
			wantErr: false,
		},
		{
			name: "missing app_id",
			config: map[string]interface{}{
				"app_secret": "secret",
				"to_user":    "openid-xxx",
			},
			wantErr: true,
		},
		{
			name: "missing app_secret",
			config: map[string]interface{}{
				"app_id":  "wx1234567890",
				"to_user": "openid-xxx",
			},
			wantErr: true,
		},
		{
			name: "missing to_user",
			config: map[string]interface{}{
				"app_id":     "wx1234567890",
				"app_secret": "secret",
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := provider.Validate(testCase.config)
			if (err != nil) != testCase.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, testCase.wantErr)
			}
		})
	}
}

func TestWeChatMPProvider_Type(t *testing.T) {
	provider := NewWeChatMPProvider()
	if provider.Type() != model.ChannelTypeWeChatMP {
		t.Fatalf("Type() = %v, want %v", provider.Type(), model.ChannelTypeWeChatMP)
	}
}
