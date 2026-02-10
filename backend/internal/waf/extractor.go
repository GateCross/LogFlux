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
		return nil, fmt.Errorf("package path is required")
	}
	if strings.TrimSpace(targetDir) == "" {
		return nil, fmt.Errorf("target dir is required")
	}

	cleanTargetDir := filepath.Clean(targetDir)
	if err := os.MkdirAll(cleanTargetDir, 0o755); err != nil {
		return nil, fmt.Errorf("create target dir failed: %w", err)
	}

	ext := detectPackageExt(packagePath)
	if ext == "" {
		return nil, fmt.Errorf("unsupported package extension")
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
		return nil, fmt.Errorf("open zip failed: %w", err)
	}
	defer reader.Close()

	result := &ExtractResult{ExtractPath: targetDir}
	for _, file := range reader.File {
		if err := validateArchiveEntry(file.Name, targetDir); err != nil {
			return nil, err
		}
		if file.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("symlink entry is not allowed: %s", file.Name)
		}

		targetPath := filepath.Join(targetDir, file.Name)
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return nil, fmt.Errorf("create dir failed: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return nil, fmt.Errorf("create parent dir failed: %w", err)
		}

		srcFile, err := file.Open()
		if err != nil {
			return nil, fmt.Errorf("open zip entry failed: %w", err)
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
		return nil, fmt.Errorf("open package failed: %w", err)
	}
	defer packageFile.Close()

	gzipReader, err := gzip.NewReader(packageFile)
	if err != nil {
		return nil, fmt.Errorf("create gzip reader failed: %w", err)
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
			return nil, fmt.Errorf("read tar entry failed: %w", err)
		}

		if err := validateArchiveEntry(header.Name, targetDir); err != nil {
			return nil, err
		}

		targetPath := filepath.Join(targetDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, 0o755); err != nil {
				return nil, fmt.Errorf("create dir failed: %w", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
				return nil, fmt.Errorf("create parent dir failed: %w", err)
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
			return nil, fmt.Errorf("symlink entry is not allowed: %s", header.Name)
		default:
			continue
		}
	}

	return result, nil
}

func validateArchiveEntry(entryName, targetDir string) error {
	if strings.TrimSpace(entryName) == "" {
		return fmt.Errorf("archive entry name is empty")
	}

	if filepath.IsAbs(entryName) {
		return fmt.Errorf("absolute path is not allowed: %s", entryName)
	}

	cleanName := filepath.Clean(entryName)
	if cleanName == "." || strings.HasPrefix(cleanName, "..") {
		return fmt.Errorf("path traversal detected: %s", entryName)
	}

	targetPath := filepath.Clean(filepath.Join(targetDir, cleanName))
	relativePath, err := filepath.Rel(targetDir, targetPath)
	if err != nil {
		return fmt.Errorf("resolve entry path failed: %w", err)
	}
	if strings.HasPrefix(relativePath, "..") || filepath.IsAbs(relativePath) {
		return fmt.Errorf("entry escapes target dir: %s", entryName)
	}
	return nil
}

func writeExtractedFile(targetPath string, source io.Reader) (int64, error) {
	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, fmt.Errorf("create target file failed: %w", err)
	}
	defer targetFile.Close()

	writtenBytes, err := io.Copy(targetFile, source)
	if err != nil {
		return 0, fmt.Errorf("write target file failed: %w", err)
	}
	return writtenBytes, nil
}

func validateExtractLimits(result *ExtractResult, options normalizedExtractOptions) error {
	if result.FileCount > options.maxFiles {
		return fmt.Errorf("too many extracted files: %d > %d", result.FileCount, options.maxFiles)
	}
	if result.TotalBytes > options.maxTotalBytes {
		return fmt.Errorf("extracted bytes exceed limit: %d > %d", result.TotalBytes, options.maxTotalBytes)
	}
	return nil
}
