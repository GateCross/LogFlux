package waf

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultExtractMaxFiles      = 5000
	DefaultExtractMaxTotalBytes = int64(512 * 1024 * 1024)
)

type ExtractOptions struct {
	MaxFiles      int
	MaxTotalBytes int64
}

type ExtractResult struct {
	FileCount   int
	TotalBytes  int64
	ExtractPath string
}

func ExtractPackage(packagePath, targetDir string, options ExtractOptions) (*ExtractResult, error) {
	if strings.TrimSpace(packagePath) == "" {
		return nil, fmt.Errorf("包路径不能为空")
	}
	if strings.TrimSpace(targetDir) == "" {
		return nil, fmt.Errorf("目标目录不能为空")
	}

	cleanTargetDir := filepath.Clean(targetDir)
	if err := os.MkdirAll(cleanTargetDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建目标目录失败: %w", err)
	}

	ext := detectPackageExt(packagePath)
	if ext == "" {
		return nil, fmt.Errorf("不支持的包扩展名")
	}

	opts := normalizeExtractOptions(options)
	if ext == ".zip" {
		return extractZip(packagePath, cleanTargetDir, opts)
	}
	return extractTarGz(packagePath, cleanTargetDir, opts)
}

type normalizedExtractOptions struct {
	maxFiles      int
	maxTotalBytes int64
}

func normalizeExtractOptions(options ExtractOptions) normalizedExtractOptions {
	maxFiles := options.MaxFiles
	if maxFiles <= 0 {
		maxFiles = DefaultExtractMaxFiles
	}
	maxTotalBytes := options.MaxTotalBytes
	if maxTotalBytes <= 0 {
		maxTotalBytes = DefaultExtractMaxTotalBytes
	}

	return normalizedExtractOptions{
		maxFiles:      maxFiles,
		maxTotalBytes: maxTotalBytes,
	}
}

func extractZip(packagePath, targetDir string, options normalizedExtractOptions) (*ExtractResult, error) {
	reader, err := zip.OpenReader(packagePath)
	if err != nil {
		return nil, fmt.Errorf("打开 ZIP 包失败: %w", err)
	}
	defer reader.Close()

	result := &ExtractResult{ExtractPath: targetDir}
	for _, file := range reader.File {
		if err := validateArchiveEntry(file.Name, targetDir); err != nil {
			return nil, err
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("不允许解压符号链接条目: %s", file.Name)
		}

		targetPath := filepath.Join(targetDir, file.Name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return nil, fmt.Errorf("创建目录失败: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return nil, fmt.Errorf("创建父目录失败: %w", err)
		}

		srcFile, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("打开 ZIP 条目失败: %w", err)
		}

		writtenBytes, err := writeExtractedFile(targetPath, srcFile)
		_ = srcFile.Close()
		if err != nil {
			return nil, err
		}

		result.FileCount++
		result.TotalBytes += writtenBytes
		if err := validateExtractLimits(result, options); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func extractTarGz(packagePath, targetDir string, options normalizedExtractOptions) (*ExtractResult, error) {
	packageFile, err := os.Open(packagePath)
	if err != nil {
		return nil, fmt.Errorf("打开包文件失败: %w", err)
	}
	defer packageFile.Close()

	gzipReader, err := gzip.NewReader(packageFile)
	if err != nil {
		return nil, fmt.Errorf("创建 gzip 读取器失败: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	result := &ExtractResult{ExtractPath: targetDir}

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("读取 tar 条目失败: %w", err)
		}

		if err := validateArchiveEntry(header.Name, targetDir); err != nil {
			return nil, err
		}

		targetPath := filepath.Join(targetDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return nil, fmt.Errorf("创建目录失败: %w", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return nil, fmt.Errorf("创建父目录失败: %w", err)
			}
			writtenBytes, err := writeExtractedFile(targetPath, tarReader)
			if err != nil {
				return nil, err
			}
			result.FileCount++
			result.TotalBytes += writtenBytes
			if err := validateExtractLimits(result, options); err != nil {
				return nil, err
			}
		case tar.TypeSymlink, tar.TypeLink:
			return nil, fmt.Errorf("不允许解压符号链接条目: %s", header.Name)
		default:
			continue
		}
	}

	return result, nil
}

func validateArchiveEntry(entryName, targetDir string) error {
	if strings.TrimSpace(entryName) == "" {
		return fmt.Errorf("归档条目名称为空")
	}

	if filepath.IsAbs(entryName) {
		return fmt.Errorf("不允许使用绝对路径: %s", entryName)
	}

	cleanName := filepath.Clean(entryName)
	if cleanName == "." || strings.HasPrefix(cleanName, "..") {
		return fmt.Errorf("检测到路径穿越: %s", entryName)
	}

	targetPath := filepath.Clean(filepath.Join(targetDir, cleanName))
	relativePath, err := filepath.Rel(targetDir, targetPath)
	if err != nil {
		return fmt.Errorf("解析条目路径失败: %w", err)
	}
	if strings.HasPrefix(relativePath, "..") || filepath.IsAbs(relativePath) {
		return fmt.Errorf("条目超出目标目录: %s", entryName)
	}
	return nil
}

func writeExtractedFile(targetPath string, source io.Reader) (int64, error) {
	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer targetFile.Close()

	writtenBytes, err := io.Copy(targetFile, source)
	if err != nil {
		return 0, fmt.Errorf("写入目标文件失败: %w", err)
	}
	return writtenBytes, nil
}

func validateExtractLimits(result *ExtractResult, options normalizedExtractOptions) error {
	if result.FileCount > options.maxFiles {
		return fmt.Errorf("解压文件数量超出限制: %d > %d", result.FileCount, options.maxFiles)
	}
	if result.TotalBytes > options.maxTotalBytes {
		return fmt.Errorf("解压总大小超出限制: %d > %d", result.TotalBytes, options.maxTotalBytes)
	}
	return nil
}
