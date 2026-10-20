package caddy

import (
	"strings"
	"testing"

	commonmodel "logflux/model/common"
	wafmodel "logflux/model/waf"
)

// 站点选择存进 policy.Config 后必须能原样读回。
//
// 写入时是 []string，从库里读回时经 jsonb 解析成 []interface{}，
// 两条路径都要覆盖，否则保存后刷新页面站点选择会丢。
func TestResolveSimpleWafSiteAddresses(t *testing.T) {
	imported := []string{"a.example.com", "b.example.com"}

	cases := []struct {
		name   string
		policy *wafmodel.WafPolicy
		want   []string
	}{
		{
			name:   "无策略时回退 Caddyfile 现状",
			policy: nil,
			want:   imported,
		},
		{
			name:   "Config 为空时回退 Caddyfile 现状",
			policy: &wafmodel.WafPolicy{},
			want:   imported,
		},
		{
			name: "写入形态 []string",
			policy: &wafmodel.WafPolicy{Config: commonmodel.JSONMap{
				simpleWafSiteAddressesKey: []string{"a.example.com"},
			}},
			want: []string{"a.example.com"},
		},
		{
			name: "读库形态 []interface{}",
			policy: &wafmodel.WafPolicy{Config: commonmodel.JSONMap{
				simpleWafSiteAddressesKey: []interface{}{"a.example.com", "b.example.com"},
			}},
			want: []string{"a.example.com", "b.example.com"},
		},
		{
			name: "空选择回退 Caddyfile 现状",
			policy: &wafmodel.WafPolicy{Config: commonmodel.JSONMap{
				simpleWafSiteAddressesKey: []interface{}{},
			}},
			want: imported,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveSimpleWafSiteAddresses(tc.policy, imported)
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}
