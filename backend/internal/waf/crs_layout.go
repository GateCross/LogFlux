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
// coraza-caddy 在 load_owasp_crs 开启时，会把 coreruleset.FS（go:embed rules）
// 与 OSFS 合并作为规则根文件系统。合并后的 FS 按以下布局提供文件：
//
//	rules/@coraza.conf-recommended
//	rules/@crs-setup.conf.example
//	rules/@owasp_crs/*.conf
//	rules/@owasp_crs/*.data
//
// 其中 @ 是 embed 路径的字面量，并非特殊语法。mergefs 的 Open 会依次尝试
// 各个子文件系统，OSFS 兜底且直接使用 os.Open，因此使用文件系统绝对路径的
// Include 可以读到本项目自己管理的规则文件，从而绕开编译期内嵌的那份规则。
const (
	// CRSRulesDirName 是规则根目录名，对应 coreruleset.FS 中的 rules。
	CRSRulesDirName = "rules"

	// CRSOWASPDirName 是 OWASP CRS 规则目录名，对应内嵌布局中的 @owasp_crs。
	CRSOWASPDirName = "@owasp_crs"

	// CRSCorazaConfigName 是 Coraza 推荐基线配置文件名。
	CRSCorazaConfigName = "@coraza.conf-recommended"

	// CRSSetupConfigName 是 CRS setup 配置文件名。
	CRSSetupConfigName = "@crs-setup.conf.example"

	// crsSourceRulesDirName 是 CRS 官方发布包中的规则目录名。
	crsSourceRulesDirName = "rules"

	// crsSourceSetupConfigName 是 CRS 官方发布包顶层的 setup 配置文件名。
	crsSourceSetupConfigName = "crs-setup.conf.example"

	// crsSourceRootPrefix 是官方 tarball/zip 解压后的顶层目录名前缀。
	crsSourceRootPrefix = "coreruleset-"
)

// CRSLayout 描述一份已重构为内嵌布局的 CRS 规则树。
type CRSLayout struct {
	// Root 是规则根目录，其中包含 rules 子目录。
	Root string

	// RulesDir 是 rules 目录绝对路径。
	RulesDir string

	// OWASPDir 是 @owasp_crs 目录绝对路径。
	OWASPDir string

	// RuleFiles 是排序后的 .conf 规则文件绝对路径列表。
	RuleFiles []string

	// DataFiles 是排序后的 .data 数据文件绝对路径列表。
	DataFiles []string
}

// CorazaConfigPath 返回 Coraza 基线配置文件路径，不存在时返回空串。
func (layout *CRSLayout) CorazaConfigPath() string {
	if layout == nil {
		return ""
	}
	return existingFileOrEmpty(filepath.Join(layout.RulesDir, CRSCorazaConfigName))
}

// SetupConfigPath 返回 CRS setup 配置文件路径，不存在时返回空串。
func (layout *CRSLayout) SetupConfigPath() string {
	if layout == nil {
		return ""
	}
	return existingFileOrEmpty(filepath.Join(layout.RulesDir, CRSSetupConfigName))
}

// HasRules 判断布局中是否包含可加载的规则文件。
func (layout *CRSLayout) HasRules() bool {
	return layout != nil && len(layout.RuleFiles) > 0
}

// BuildCRSLayout 把一份解压后的 CRS 发布目录重构成内嵌文件系统的布局。
//
// 官方发布包解压后结构为：
//
//	coreruleset-<version>/rules/*.conf
//	coreruleset-<version>/rules/*.data
//	coreruleset-<version>/crs-setup.conf.example
//
// 重构后在 root 下生成：
//
//	rules/@coraza.conf-recommended   （由调用方按需补充）
//	rules/@crs-setup.conf.example
//	rules/@owasp_crs/*.conf
//	rules/@owasp_crs/*.data
//
// 规则内部通过 @pmFromFile php-variables.data 这类相对文件名引用数据文件，
// 因此 .conf 与 .data 必须位于同一目录，重构时二者统一放进 @owasp_crs。
func BuildCRSLayout(sourceDir, root string) (*CRSLayout, error) {
	trimmedSource := strings.TrimSpace(sourceDir)
	trimmedRoot := strings.TrimSpace(root)
	if trimmedSource == "" {
		return nil, fmt.Errorf("CRS 源目录为空")
	}
	if trimmedRoot == "" {
		return nil, fmt.Errorf("CRS 目标目录为空")
	}

	sourceRoot, err := locateCRSSourceRoot(trimmedSource)
	if err != nil {
		return nil, err
	}

	sourceRulesDir := filepath.Join(sourceRoot, crsSourceRulesDirName)
	if !dirExists(sourceRulesDir) {
		return nil, fmt.Errorf("CRS 规则目录不存在: %s", sourceRulesDir)
	}

	cleanRoot := filepath.Clean(trimmedRoot)
	rulesDir := filepath.Join(cleanRoot, CRSRulesDirName)
	owaspDir := filepath.Join(rulesDir, CRSOWASPDirName)

	if err := os.MkdirAll(owaspDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建 CRS 布局目录失败: %w", err)
	}

	if err := copyCRSRules(sourceRulesDir, owaspDir); err != nil {
		return nil, err
	}

	setupSource := filepath.Join(sourceRoot, crsSourceSetupConfigName)
	if fileExists(setupSource) {
		if err := copyFileTo(setupSource, filepath.Join(rulesDir, CRSSetupConfigName)); err != nil {
			return nil, err
		}
	}

	layout := &CRSLayout{Root: cleanRoot, RulesDir: rulesDir, OWASPDir: owaspDir}
	ruleFiles, err := listFilesByExt(owaspDir, ".conf")
	if err != nil {
		return nil, err
	}
	dataFiles, err := listFilesByExt(owaspDir, ".data")
	if err != nil {
		return nil, err
	}
	layout.RuleFiles = ruleFiles
	layout.DataFiles = dataFiles

	if !layout.HasRules() {
		return nil, fmt.Errorf("CRS 规则文件为空: %s", sourceRulesDir)
	}

	return layout, nil
}

// locateCRSSourceRoot 在给定目录中定位 CRS 官方发布包的根目录。
//
// 兼容三种输入：发布包根目录本身、包含 coreruleset-<version> 的上级目录、
// 以及直接指向 rules 的目录（返回其父目录）。
func locateCRSSourceRoot(sourceDir string) (string, error) {
	candidates := []string{sourceDir}

	// 传入路径本身就是 rules 目录时，其父目录即发布包根目录。
	if filepath.Base(filepath.Clean(sourceDir)) == crsSourceRulesDirName {
		candidates = append(candidates, filepath.Dir(filepath.Clean(sourceDir)))
	}

	entries, err := os.ReadDir(sourceDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if strings.HasPrefix(entry.Name(), crsSourceRootPrefix) {
				candidates = append(candidates, filepath.Join(sourceDir, entry.Name()))
			}
		}
	}

	for _, candidate := range candidates {
		if dirExists(filepath.Join(candidate, crsSourceRulesDirName)) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("未找到 CRS 规则目录: %s", sourceDir)
}

// copyCRSRules 拷贝规则与数据文件到目标目录。
//
// 仅拷贝 .conf 与 .data，跳过 .example（例如 REQUEST-900-EXCLUSION-RULES-BEFORE-CRS.conf.example，
// 它们是给用户参考的模板，默认不参与加载）以及测试目录。
func copyCRSRules(sourceRulesDir, targetDir string) error {
	entries, err := os.ReadDir(sourceRulesDir)
	if err != nil {
		return fmt.Errorf("读取 CRS 规则目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".conf") && !strings.HasSuffix(name, ".data") {
			continue
		}
		// .conf.example 会被 HasSuffix(".conf") 排除，这里显式跳过以示意图。
		if strings.HasSuffix(name, ".example") {
			continue
		}

		if err := copyFileTo(filepath.Join(sourceRulesDir, name), filepath.Join(targetDir, name)); err != nil {
			return err
		}
	}

	return nil
}

func copyFileTo(sourcePath, targetPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	targetPath += ".tmp"
	tempFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}

	if _, err := io.Copy(tempFile, sourceFile); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(targetPath)
		return fmt.Errorf("拷贝文件失败: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(targetPath)
		return fmt.Errorf("关闭目标文件失败: %w", err)
	}

	finalPath := strings.TrimSuffix(targetPath, ".tmp")
	if err := os.Rename(targetPath, finalPath); err != nil {
		_ = os.Remove(targetPath)
		return fmt.Errorf("替换目标文件失败: %w", err)
	}
	return nil
}

func listFilesByExt(dir, ext string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !strings.HasSuffix(entry.Name(), ext) {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}
	sort.Strings(files)
	return files, nil
}

func existingFileOrEmpty(path string) string {
	if fileExists(path) {
		return path
	}
	return ""
}

// fileExists 判断路径是否为已存在的普通文件。
func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	stat, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return false
	}
	return !stat.IsDir()
}

// linkOrCopyCRSRules 把发布目录的规则树对接到布局根目录。
//
// 优先使用符号链接以节省空间与拷贝耗时；在不支持符号链接的平台上退化为拷贝。
func linkOrCopyCRSRules(releaseDir, layoutRoot string) error {
	sourceRoot, err := locateCRSSourceRoot(releaseDir)
	if err != nil {
		return err
	}

	rulesDir := filepath.Join(layoutRoot, CRSRulesDirName)
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		return fmt.Errorf("创建布局目录失败: %w", err)
	}

	sourceRulesDir := filepath.Join(sourceRoot, crsSourceRulesDirName)
	owaspDir := filepath.Join(rulesDir, CRSOWASPDirName)
	if err := linkOrCopyDir(sourceRulesDir, owaspDir); err != nil {
		return err
	}

	setupSource := filepath.Join(sourceRoot, crsSourceSetupConfigName)
	if fileExists(setupSource) {
		if err := copyFileTo(setupSource, filepath.Join(rulesDir, CRSSetupConfigName)); err != nil {
			return err
		}
	}

	return nil
}

// linkOrCopyDir 把源目录对接到目标目录：能建符号链接就建，否则逐个拷贝文件。
func linkOrCopyDir(sourceDir, targetDir string) error {
	// 目标目录必须不存在才能建符号链接，父目录需提前备好。
	if err := os.MkdirAll(filepath.Dir(targetDir), 0o755); err != nil {
		return fmt.Errorf("创建父目录失败: %w", err)
	}
	_ = os.RemoveAll(targetDir)

	if err := os.Symlink(sourceDir, targetDir); err == nil {
		return nil
	}

	// 符号链接不可用（例如 Windows 无特权）时退化为逐个拷贝文件。
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("读取源目录失败: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".conf") && !strings.HasSuffix(name, ".data") {
			continue
		}
		if strings.HasSuffix(name, ".example") {
			continue
		}
		if err := copyFileTo(filepath.Join(sourceDir, name), filepath.Join(targetDir, name)); err != nil {
			return err
		}
	}

	return nil
}
