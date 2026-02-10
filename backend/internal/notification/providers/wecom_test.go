package providers

import (
	"logflux/model"
	"testing"
)

func TestWeComProvider_Validate(t *testing.T) {
	provider := NewWeComProvider()

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"webhook_url": "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test-key",
			},
			wantErr: false,
		},
		{
			name: "missing webhook_url",
			config: map[string]interface{}{
				"webhook_url": "",
			},
			wantErr: true,
		},
		{
			name: "invalid webhook_url",
			config: map[string]interface{}{
				"webhook_url": "qyapi.weixin.qq.com/cgi-bin/webhook/send?key=test-key",
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

func TestWeComProvider_Type(t *testing.T) {
	provider := NewWeComProvider()
	if provider.Type() != model.ChannelTypeWeCom {
		t.Fatalf("Type() = %v, want %v", provider.Type(), model.ChannelTypeWeCom)
	}
}
