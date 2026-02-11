package caddy

import (
	"testing"

	"logflux/model"
)

func TestExtractVersionFromSourceURL(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		output string
	}{
		{
			name:   "github refs tags",
			input:  "https://codeload.github.com/coreruleset/coreruleset/tar.gz/refs/tags/v4.23.0",
			output: "v4.23.0",
		},
		{
			name:   "github release download",
			input:  "https://github.com/coreruleset/coreruleset/releases/download/v4.22.0/coreruleset-v4.22.0.tar.gz",
			output: "v4.22.0",
		},
		{
			name:   "semver in filename",
			input:  "https://mirror.example.com/coreruleset-v4.21.1.tar.gz",
			output: "v4.21.1",
		},
		{
			name:   "branch like main with suffix",
			input:  "https://mirror.example.com/main_1770792061.tar.gz",
			output: "main_1770792061",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got := extractVersionFromSourceURL(testCase.input)
			if got != testCase.output {
				t.Fatalf("unexpected version, want=%q got=%q", testCase.output, got)
			}
		})
	}
}

func TestIsBranchLikeVersion(t *testing.T) {
	branchLikeCases := []string{"main", "master", "HEAD", "main_1770792061", "latest"}
	for _, value := range branchLikeCases {
		if !isBranchLikeVersion(value) {
			t.Fatalf("expected branch-like version: %s", value)
		}
	}

	nonBranchCases := []string{"v4.23.0", "v4.23.0-rc1", "2026.02.11"}
	for _, value := range nonBranchCases {
		if isBranchLikeVersion(value) {
			t.Fatalf("expected non branch-like version: %s", value)
		}
	}
}

func TestIsOfficialCRSSource(t *testing.T) {
	official := &model.WafSource{Kind: "crs", URL: "https://github.com/coreruleset/coreruleset/archive/refs/heads/main.tar.gz"}
	if !isOfficialCRSSource(official) {
		t.Fatalf("expected official CRS source")
	}

	officialByMeta := &model.WafSource{Kind: "crs", URL: "https://mirror.example.com/crs.tar.gz", Meta: model.JSONMap{"repo": "https://github.com/coreruleset/coreruleset"}}
	if !isOfficialCRSSource(officialByMeta) {
		t.Fatalf("expected official CRS source by meta")
	}

	custom := &model.WafSource{Kind: "crs", URL: "https://example.com/security-rules.tar.gz"}
	if isOfficialCRSSource(custom) {
		t.Fatalf("expected custom source not official")
	}
}

func TestBuildCRSReleaseTagDownloadURL(t *testing.T) {
	got := buildCRSReleaseTagDownloadURL("v4.23.0")
	want := "https://codeload.github.com/coreruleset/coreruleset/tar.gz/refs/tags/v4.23.0"
	if got != want {
		t.Fatalf("unexpected download url, want=%q got=%q", want, got)
	}
}

func TestResolveCRSSyncTargetFallbackForCustomSource(t *testing.T) {
	helper := &wafLogicHelper{}
	source := &model.WafSource{
		Name: "custom-crs",
		Kind: "crs",
		URL:  "https://mirror.example.com/main_1770792061.tar.gz",
	}

	downloadURL, version := helper.resolveCRSSyncTarget(source)
	if downloadURL != source.URL {
		t.Fatalf("unexpected download url, want=%q got=%q", source.URL, downloadURL)
	}
	if version != "main_1770792061" {
		t.Fatalf("unexpected version, want=%q got=%q", "main_1770792061", version)
	}
}
