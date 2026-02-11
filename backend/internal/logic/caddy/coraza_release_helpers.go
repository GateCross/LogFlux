package caddy

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"logflux/model"
)

const defaultCorazaReleaseAPI = "https://api.github.com/repos/corazawaf/coraza-caddy/releases/latest"

var corazaModuleVersionPattern = regexp.MustCompile(`github\.com/corazawaf/coraza-caddy(?:/v\d+)?@([A-Za-z0-9._+\-~]+)`)
var semverPattern = regexp.MustCompile(`v\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.\-]+)?`)

type githubLatestReleaseResp struct {
	TagName string `json:"tag_name"`
}

func (helper *wafLogicHelper) corazaCurrentVersion() string {
	configuredVersion := strings.TrimSpace(helper.svcCtx.Config.Waf.CorazaCurrentVersion)
	if configuredVersion != "" {
		return configuredVersion
	}

	envVersion := strings.TrimSpace(os.Getenv("CORAZA_CURRENT_VERSION"))
	if envVersion != "" {
		return envVersion
	}

	fileVersion, _ := os.ReadFile("/app/etc/coraza-current-version")
	trimmedFileVersion := strings.TrimSpace(string(fileVersion))
	if trimmedFileVersion != "" {
		return trimmedFileVersion
	}

	detectedVersion, detectErr := helper.detectCorazaCurrentVersion()
	if detectErr != nil {
		helper.logger.Errorf("detect coraza current version failed: %v", detectErr)
	}
	return strings.TrimSpace(detectedVersion)
}

func (helper *wafLogicHelper) detectCorazaCurrentVersion() (string, error) {
	commandCandidates := [][]string{
		{"caddy", "list-modules", "--versions"},
		{"/usr/bin/caddy", "list-modules", "--versions"},
		{"caddy", "build-info"},
		{"/usr/bin/caddy", "build-info"},
	}

	var lastErr error
	for _, command := range commandCandidates {
		if len(command) == 0 {
			continue
		}
		version, err := detectCorazaVersionByCommand(helper.ctx, command[0], command[1:]...)
		if err != nil {
			if isCommandNotFoundError(err) {
				continue
			}
			lastErr = err
			continue
		}
		if version != "" {
			return version, nil
		}
	}

	if lastErr != nil {
		return "", lastErr
	}
	return "", nil
}

func detectCorazaVersionByCommand(parentCtx context.Context, name string, args ...string) (string, error) {
	baseCtx := parentCtx
	if baseCtx == nil {
		baseCtx = context.Background()
	}
	timeoutCtx, cancel := context.WithTimeout(baseCtx, 3*time.Second)
	defer cancel()

	command := exec.CommandContext(timeoutCtx, name, args...)
	outputBytes, err := command.CombinedOutput()
	outputText := strings.TrimSpace(string(outputBytes))
	if err != nil {
		if outputText != "" {
			return "", fmt.Errorf("exec %s %s failed: %w, output=%s", name, strings.Join(args, " "), err, outputText)
		}
		return "", fmt.Errorf("exec %s %s failed: %w", name, strings.Join(args, " "), err)
	}

	return extractCorazaVersionFromText(outputText), nil
}

func extractCorazaVersionFromText(rawOutput string) string {
	output := strings.TrimSpace(rawOutput)
	if output == "" {
		return ""
	}

	if matches := corazaModuleVersionPattern.FindStringSubmatch(output); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}

	for _, line := range strings.Split(output, "\n") {
		lineText := strings.TrimSpace(line)
		if lineText == "" {
			continue
		}
		if !strings.Contains(strings.ToLower(lineText), "coraza") {
			continue
		}
		if matched := semverPattern.FindString(lineText); matched != "" {
			return strings.TrimSpace(matched)
		}
	}

	return ""
}

func isCommandNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	messageText := strings.ToLower(err.Error())
	if strings.Contains(messageText, "executable file not found") {
		return true
	}
	if strings.Contains(messageText, "no such file or directory") {
		return true
	}
	return false
}

func (helper *wafLogicHelper) corazaReleaseAPI() string {
	releaseAPI := strings.TrimSpace(helper.svcCtx.Config.Waf.CorazaReleaseAPI)
	if releaseAPI == "" {
		releaseAPI = strings.TrimSpace(os.Getenv("CORAZA_RELEASE_API"))
	}
	if releaseAPI == "" {
		releaseAPI = defaultCorazaReleaseAPI
	}
	return releaseAPI
}

func (helper *wafLogicHelper) corazaCheckProxy() string {
	proxyURL := strings.TrimSpace(helper.svcCtx.Config.Waf.CorazaCheckProxy)
	if proxyURL != "" {
		return proxyURL
	}
	return strings.TrimSpace(os.Getenv("CORAZA_CHECK_PROXY"))
}

func (helper *wafLogicHelper) fetchCorazaLatestReleaseVersion() (string, error) {
	releaseAPI := helper.corazaReleaseAPI()
	proxyURL := helper.corazaCheckProxy()
	timeoutSec := helper.svcCtx.Config.Waf.FetchTimeoutSec

	version, err := fetchGithubLatestReleaseTag(releaseAPI, timeoutSec, proxyURL)
	if err != nil && proxyURL != "" {
		helper.logger.Errorf("coraza release check by proxy failed, fallback direct: proxy=%s err=%v", proxyURL, err)
		version, err = fetchGithubLatestReleaseTag(releaseAPI, timeoutSec, "")
	}
	if err != nil {
		return "", err
	}
	return version, nil
}

func fetchGithubLatestReleaseTag(releaseAPI string, timeoutSec int, proxyURL string) (string, error) {
	parsedURL, err := url.Parse(strings.TrimSpace(releaseAPI))
	if err != nil {
		return "", fmt.Errorf("invalid coraza release api url: %w", err)
	}
	if parsedURL.Scheme != "https" {
		return "", fmt.Errorf("coraza release api only supports https")
	}

	timeout := time.Duration(timeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 60 * time.Second
	}

	transport := &http.Transport{TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12}}
	if strings.TrimSpace(proxyURL) != "" {
		parsedProxyURL, proxyErr := url.Parse(strings.TrimSpace(proxyURL))
		if proxyErr != nil {
			return "", fmt.Errorf("invalid coraza check proxy url: %w", proxyErr)
		}
		if parsedProxyURL.Scheme != "http" && parsedProxyURL.Scheme != "https" {
			return "", fmt.Errorf("coraza check proxy scheme must be http or https")
		}
		transport.Proxy = http.ProxyURL(parsedProxyURL)
	}

	httpClient := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	request, err := http.NewRequest(http.MethodGet, parsedURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("create coraza release request failed: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "logflux-coraza-version-checker")

	response, err := httpClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("request coraza release failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(response.Body, 1024))
		bodyText := strings.TrimSpace(string(bodyBytes))
		if bodyText != "" {
			return "", fmt.Errorf("request coraza release failed: status=%d body=%s", response.StatusCode, bodyText)
		}
		return "", fmt.Errorf("request coraza release failed: status=%d", response.StatusCode)
	}

	var payload githubLatestReleaseResp
	if err := json.NewDecoder(io.LimitReader(response.Body, 1024*1024)).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode coraza release failed: %w", err)
	}

	tag := strings.TrimSpace(payload.TagName)
	if tag == "" {
		return "", fmt.Errorf("coraza release tag is empty")
	}

	return tag, nil
}

func latestEngineCheckVersion(job *model.WafUpdateJob) string {
	if job == nil || len(job.Meta) == 0 {
		return ""
	}
	if rawVersion, ok := job.Meta["latestVersion"]; ok {
		return strings.TrimSpace(fmt.Sprint(rawVersion))
	}
	return ""
}
