package waf

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	DefaultMaxPackageBytes int64 = 100 * 1024 * 1024
)

type VerifyOptions struct {
	AllowedExt      []string
	MaxPackageBytes int64
	ExpectedSHA256  string
}

type VerifyResult struct {
	SizeBytes int64
	SHA256    string
	Ext       string
}

func VerifyPackage(packagePath string, options VerifyOptions) (*VerifyResult, error) {
	if strings.TrimSpace(packagePath) == "" {
		return nil, fmt.Errorf("package path is required")
	}

	allowedExt := normalizeAllowedExt(options.AllowedExt)
	if len(allowedExt) == 0 {
		allowedExt = []string{".tar.gz", ".zip"}
	}

	packageExt := detectPackageExt(packagePath)
	if packageExt == "" || !slices.Contains(allowedExt, packageExt) {
		return nil, fmt.Errorf("unsupported package extension: %s", filepath.Base(packagePath))
	}

	fileInfo, err := os.Stat(packagePath)
	if err != nil {
		return nil, fmt.Errorf("stat package failed: %w", err)
	}

	maxPackageBytes := options.MaxPackageBytes
	if maxPackageBytes <= 0 {
		maxPackageBytes = DefaultMaxPackageBytes
	}
	if fileInfo.Size() > maxPackageBytes {
		return nil, fmt.Errorf("package too large: %d > %d", fileInfo.Size(), maxPackageBytes)
	}

	hash, err := calculateFileSHA256(packagePath)
	if err != nil {
		return nil, err
	}

	expectedHash := normalizeHash(options.ExpectedSHA256)
	if expectedHash != "" && hash != expectedHash {
		return nil, fmt.Errorf("sha256 mismatch: expected %s, got %s", expectedHash, hash)
	}

	return &VerifyResult{
		SizeBytes: fileInfo.Size(),
		SHA256:    hash,
		Ext:       packageExt,
	}, nil
}

func calculateFileSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("open package failed: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("hash package failed: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func normalizeAllowedExt(extensions []string) []string {
	if len(extensions) == 0 {
		return nil
	}

	unique := make(map[string]struct{}, len(extensions))
	results := make([]string, 0, len(extensions))
	for _, extension := range extensions {
		normalized := strings.ToLower(strings.TrimSpace(extension))
		if normalized == "" {
			continue
		}
		if !strings.HasPrefix(normalized, ".") {
			normalized = "." + normalized
		}
		if _, exists := unique[normalized]; exists {
			continue
		}
		unique[normalized] = struct{}{}
		results = append(results, normalized)
	}
	return results
}

func normalizeHash(hash string) string {
	return strings.ToLower(strings.TrimSpace(hash))
}

func detectPackageExt(filePath string) string {
	lowerName := strings.ToLower(filepath.Base(filePath))
	switch {
	case strings.HasSuffix(lowerName, ".tar.gz"):
		return ".tar.gz"
	case strings.HasSuffix(lowerName, ".zip"):
		return ".zip"
	default:
		return ""
	}
}
