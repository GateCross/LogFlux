package caddy

import (
	"fmt"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"logflux/model"
)

const defaultCRSReleaseAPI = "https://api.github.com/repos/coreruleset/coreruleset/releases/latest"

var crsTagPathPattern = regexp.MustCompile(`/refs/tags/([^/?#]+)`)
var githubReleaseDownloadTagPattern = regexp.MustCompile(`/releases/download/([^/]+)/`)
var branchLikeVersionPattern = regexp.MustCompile(`(?i)^(main|master|head|latest)([_-].*)?$`)

func (helper *wafLogicHelper) crsCurrentVersion() string {
	if helper == nil || helper.svcCtx == nil || helper.svcCtx.DB == nil {
		return ""
	}

	var release model.WafRelease
	if err := helper.svcCtx.DB.WithContext(helper.ctx).
		Where("kind = ? AND status = ?", wafKindCRS, wafReleaseStatusActive).
		Order("updated_at desc, id desc").
		First(&release).Error; err == nil {
		if version := strings.TrimSpace(release.Version); version != "" {
			return version
		}
	}

	if helper.store != nil {
		if targetPath, err := helper.store.LinkTarget(helper.store.CurrentLinkPath()); err == nil {
			if version := strings.TrimSpace(filepath.Base(targetPath)); version != "" && version != "." && version != string(filepath.Separator) {
				return version
			}
		}
	}

	detectedVersion, detectErr := helper.detectCRSCurrentVersion()
	if detectErr != nil {
		helper.logger.Errorf("检测 CRS 当前版本失败: %v", detectErr)
	}
	return strings.TrimSpace(detectedVersion)
}

func (helper *wafLogicHelper) resolveCRSSyncTarget(source *model.WafSource) (string, string) {
	if source == nil {
		return "", ""
	}

	downloadURL := strings.TrimSpace(source.URL)
	version := extractVersionFromSourceURL(downloadURL)

	if isOfficialCRSSource(source) && (version == "" || isBranchLikeVersion(version)) {
		latestTag, err := helper.fetchCRSLatestReleaseTag(strings.TrimSpace(source.ProxyURL))
		if err != nil {
			helper.logger.Errorf("解析 CRS 最新版本失败，回退使用源地址: source=%s url=%s err=%v", strings.TrimSpace(source.Name), downloadURL, err)
		} else if latestTag != "" {
			return buildCRSReleaseTagDownloadURL(latestTag), latestTag
		}
	}

	if version == "" {
		version = deriveVersionFromURL(downloadURL)
	}

	return downloadURL, version
}

func (helper *wafLogicHelper) fetchCRSLatestReleaseTag(proxyURL string) (string, error) {
	timeoutSec := helper.svcCtx.Config.Waf.FetchTimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = 180
	}

	tag, err := fetchGithubLatestReleaseTag(defaultCRSReleaseAPI, timeoutSec, proxyURL)
	if err != nil && proxyURL != "" {
		helper.logger.Errorf("通过代理检查 CRS 版本失败，回退直连: proxy=%s err=%v", proxyURL, err)
		tag, err = fetchGithubLatestReleaseTag(defaultCRSReleaseAPI, timeoutSec, "")
	}
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(tag), nil
}

func buildCRSReleaseTagDownloadURL(tag string) string {
	trimmedTag := strings.TrimSpace(tag)
	if trimmedTag == "" {
		return ""
	}

	escapedTag := url.PathEscape(trimmedTag)
	return fmt.Sprintf("https://codeload.github.com/coreruleset/coreruleset/tar.gz/refs/tags/%s", escapedTag)
}

func extractVersionFromSourceURL(downloadURL string) string {
	parsedURL, err := url.Parse(strings.TrimSpace(downloadURL))
	if err != nil {
		return ""
	}

	cleanPath := strings.TrimSpace(parsedURL.Path)
	if cleanPath == "" {
		return ""
	}

	if matches := crsTagPathPattern.FindStringSubmatch(cleanPath); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}

	if matches := githubReleaseDownloadTagPattern.FindStringSubmatch(cleanPath); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}

	baseName := path.Base(cleanPath)
	if baseName == "." || baseName == "/" || strings.TrimSpace(baseName) == "" {
		return ""
	}

	ext := detectPackageExt(baseName)
	version := baseName
	if ext != "" {
		version = strings.TrimSuffix(baseName, ext)
	}

	if semver := semverPattern.FindString(version); semver != "" {
		return strings.TrimSpace(semver)
	}

	return strings.TrimSpace(version)
}

func isBranchLikeVersion(version string) bool {
	normalized := strings.ToLower(strings.TrimSpace(version))
	if normalized == "" {
		return false
	}

	normalized = strings.TrimPrefix(normalized, "refs_heads_")
	return branchLikeVersionPattern.MatchString(normalized)
}

func isOfficialCRSSource(source *model.WafSource) bool {
	if source == nil {
		return false
	}
	if normalizeWafKind(source.Kind) != wafKindCRS {
		return false
	}

	candidates := []string{strings.TrimSpace(source.URL)}
	if source.Meta != nil {
		if repo, ok := source.Meta["repo"]; ok {
			candidates = append(candidates, fmt.Sprint(repo))
		}
	}

	for _, candidate := range candidates {
		normalized := strings.ToLower(strings.TrimSpace(candidate))
		if normalized == "" {
			continue
		}
		if strings.Contains(normalized, "github.com/coreruleset/coreruleset") ||
			strings.Contains(normalized, "codeload.github.com/coreruleset/coreruleset") ||
			strings.Contains(normalized, "coreruleset/coreruleset") {
			return true
		}
	}

	return false
}
