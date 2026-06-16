package middleware

import "testing"

func TestIsRegionAllowed_MatchesCountryProvinceAndCity(t *testing.T) {
	tests := []struct {
		name      string
		allowList []string
		region    string
		want      bool
	}{
		{
			name:      "国家级放行",
			allowList: []string{"中国"},
			region:    "中国|四川省|绵阳市|电信|CN",
			want:      true,
		},
		{
			name:      "省级放行",
			allowList: []string{"中国/四川省"},
			region:    "中国|四川省|绵阳市|电信|CN",
			want:      true,
		},
		{
			name:      "市级放行",
			allowList: []string{"中国/四川省/绵阳市"},
			region:    "中国|四川省|绵阳市|电信|CN",
			want:      true,
		},
		{
			name:      "省级不匹配时拦截",
			allowList: []string{"中国/广东省"},
			region:    "中国|四川省|绵阳市|电信|CN",
			want:      false,
		},
		{
			name:      "市级不匹配时拦截",
			allowList: []string{"中国/四川省/成都市"},
			region:    "中国|四川省|绵阳市|电信|CN",
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			country, province, city := parseRegionParts(tt.region)
			got := isRegionAllowed(buildRegionAllowSet(tt.allowList), country, province, city)
			if got != tt.want {
				t.Fatalf("isRegionAllowed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeRegionRule(t *testing.T) {
	got := normalizeRegionRule(" / 中国 / 四川省 / 绵阳市 / ")
	if got != "中国/四川省/绵阳市" {
		t.Fatalf("normalizeRegionRule() = %q", got)
	}
}

func TestResolve_ReturnsRegionForPublicIP(t *testing.T) {
	m := NewIPRegionMiddleware(false, nil, nil)

	country, province, city := m.Resolve("125.65.97.87")
	if country == "" {
		t.Fatalf("Resolve() returned empty country, province=%q city=%q", province, city)
	}
}
