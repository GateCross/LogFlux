package shared

import "testing"

// Caddy 滚动后的归档文件名必须与活动日志区分开。
// 归档后缀同样是 .log，若按扩展名判断会被当成新文件从头采集，造成重复入库。
func TestIsActiveLogFileName(t *testing.T) {
	cases := []struct {
		name string
		want bool
	}{
		// 活动日志：应当采集
		{"access.log", true},
		{"myapp.log", true},
		{"example.com.log", true},
		{"sub-domain.log", true},

		// 滚动归档：跳过
		{"access-2026-10-26T16-00-00.000-size.log", false},
		{"access-2026-10-26T16-00-00.000-time.log", false},
		{"access-2026-10-26T16-00-00.000-size.log.gz", false},
		{"access-2026-10-26T16-00-00.000-time.log.gz", false},

		// 非日志文件
		{"access.txt", false},
		{"README", false},
		{".gitkeep", false},
		{"", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsActiveLogFileName(tc.name); got != tc.want {
				t.Fatalf("IsActiveLogFileName(%q) = %v, want %v", tc.name, got, tc.want)
			}
		})
	}
}

// 名称里本身含 size/time 但不是归档形态的文件不应被误杀。
func TestIsRolledLogFileName_DoesNotOverMatch(t *testing.T) {
	cases := []string{
		"size.log",
		"time.log",
		"file-size.log",
		"my-time.log",
		"access.log",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			if IsRolledLogFileName(name) {
				t.Fatalf("IsRolledLogFileName(%q) = true, want false", name)
			}
		})
	}
}
