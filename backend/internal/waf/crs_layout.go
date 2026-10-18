package waf

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// 内嵌 CRS 文件系统的目录布局。
//
// coraza-caddy 在 load_owasp_crs 开启时把 coreruleset.FS 与 OSFS 合并，
// 绝对路径从 OSFS 读取，从而接管内嵌规则，让在线更新真正生效。
const (
	CRSRulesDirName     = "rules"
	CRSOWASPDirName     = "@owasp_crs"
	CRSCorazaConfigName = "@coraza.conf-recommended"
	CRSSetupConfigName  = "@crs-setup.conf.example"

	crsSourceRulesDirName    = "rules"
	crsSourceSetupConfigName = "crs-setup.conf.example"
	crsSourceRootPrefix      = "coreruleset-"
)

// CRSLayout 描述已重构为内嵌同构布局的 CRS 规则树。
type CRSLayout struct {
	Root      string
	RulesDir  string
	OWASPDir  string
	RuleFiles []string
	DataFiles []string
}

func (layout *CRSLayout) CorazaConfigPath() string {
	if layout == nil {
		return ""
	}
	p := filepath.Join(layout.RulesDir, CRSCorazaConfigName)
	if FileExists(p) {
		return p
	}
	return ""
}

func (layout *CRSLayout) SetupConfigPath() string {
	if layout == nil {
		return ""
	}
	p := filepath.Join(layout.RulesDir, CRSSetupConfigName)
	if FileExists(p) {
		return p
	}
	return ""
}

func (layout *CRSLayout) HasRules() bool {
	return layout != nil && len(layout.RuleFiles) > 0
}

// BuildCRSLayout 把解压后的官方 CRS 发布包重构成 Caddy 可直接加载的布局。
//
// 优先使用符号链接（秒级、0 空间），在无特权环境退化为逐文件复制。
// 规则内部以 @pmFromFile xxx.data 引用同目录数据，二者统一放进 @owasp_crs。
func BuildCRSLayout(sourceDir, targetRoot string) (*CRSLayout, error) {
	cleanSource := filepath.Clean(strings.TrimSpace(sourceDir))
	cleanRoot := filepath.Clean(strings.TrimSpace(targetRoot))
	if cleanSource == "" || cleanSource == "." {
		return nil, fmt.Errorf("CRS 源目录为空")
	}
	if cleanRoot == "" || cleanRoot == "." {
		return nil, fmt.Errorf("CRS 目标目录为空")
	}

	sourceRoot, err := locateCRSSourceRoot(cleanSource)
	if err != nil {
		return nil, err
	}

	sourceRulesDir := filepath.Join(sourceRoot, crsSourceRulesDirName)
	if !dirExists(sourceRulesDir) {
		return nil, fmt.Errorf("CRS 规则目录不存在: %s", sourceRulesDir)
	}

	rulesDir := filepath.Join(cleanRoot, CRSRulesDirName)
	owaspDir := filepath.Join(rulesDir, CRSOWASPDirName)
	if err := linkOrCopyDir(sourceRulesDir, owaspDir); err != nil {
		return nil, err
	}

	setupSrc := filepath.Join(sourceRoot, crsSourceSetupConfigName)
	if FileExists(setupSrc) {
		if err := CopyFile(setupSrc, filepath.Join(rulesDir, CRSSetupConfigName)); err != nil {
			return nil, err
		}
	}

	ruleFiles, _ := listFilesByExt(owaspDir, ".conf")
	dataFiles, _ := listFilesByExt(owaspDir, ".data")

	layout := &CRSLayout{
		Root:      cleanRoot,
		RulesDir:  rulesDir,
		OWASPDir:  owaspDir,
		RuleFiles: ruleFiles,
		DataFiles: dataFiles,
	}
	if !layout.HasRules() {
		return nil, fmt.Errorf("CRS 规则文件为空: %s", sourceRulesDir)
	}
	return layout, nil
}

func locateCRSSourceRoot(sourceDir string) (string, error) {
	candidates := []string{sourceDir}
	if filepath.Base(sourceDir) == crsSourceRulesDirName {
		candidates = append(candidates, filepath.Dir(sourceDir))
	}
	if entries, err := os.ReadDir(sourceDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && strings.HasPrefix(e.Name(), crsSourceRootPrefix) {
				candidates = append(candidates, filepath.Join(sourceDir, e.Name()))
			}
		}
	}
	for _, c := range candidates {
		if dirExists(filepath.Join(c, crsSourceRulesDirName)) {
			return c, nil
		}
	}
	return "", fmt.Errorf("未找到 CRS 规则目录: %s", sourceDir)
}

func linkOrCopyDir(sourceDir, targetDir string) error {
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return err
	}
	_ = os.RemoveAll(targetDir)

	if err := os.Symlink(sourceDir, targetDir); err == nil {
		return nil
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if (!strings.HasSuffix(name, ".conf") && !strings.HasSuffix(name, ".data")) || strings.HasSuffix(name, ".example") {
			continue
		}
		if err := CopyFile(filepath.Join(sourceDir, name), filepath.Join(targetDir, name)); err != nil {
			return err
		}
	}
	return nil
}

// CopyFile 原子复制单个文件。
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, dst)
}

// FileExists 判断是否为已存在的普通文件。
func FileExists(path string) bool {
	info, err := os.Stat(strings.TrimSpace(path))
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func listFilesByExt(dir, ext string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ext) && !strings.HasSuffix(e.Name(), ".example") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	return files, nil
}
