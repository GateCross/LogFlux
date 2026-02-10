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
				"corp_id":     "ww1234567890",
				"corp_secret": "secret",
				"agent_id":    1000002,
				"to_user":     "zhangsan",
			},
			wantErr: false,
		},
		{
			name: "missing corp_id",
			config: map[string]interface{}{
				"corp_secret": "secret",
				"agent_id":    1000002,
				"to_user":     "zhangsan",
			},
			wantErr: true,
		},
		{
			name: "missing corp_secret",
			config: map[string]interface{}{
				"corp_id":  "ww1234567890",
				"agent_id": 1000002,
				"to_user":  "zhangsan",
			},
			wantErr: true,
		},
		{
			name: "missing agent_id",
			config: map[string]interface{}{
				"corp_id":     "ww1234567890",
				"corp_secret": "secret",
				"to_user":     "zhangsan",
			},
			wantErr: true,
		},
		{
			name: "invalid msg_type",
			config: map[string]interface{}{
				"corp_id":     "ww1234567890",
				"corp_secret": "secret",
				"agent_id":    1000002,
				"msg_type":    "image",
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
