package waf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	CurrentLinkName  = "current"
	LastGoodLinkName = "last_good"
)

type Store struct {
	BaseDir     string
	PackagesDir string
	ReleasesDir string
}

func NewStore(baseDir string) *Store {
	baseDir = strings.TrimSpace(baseDir)
	if baseDir == "" {
		baseDir = "/config/security"
	}
	baseDir = filepath.Clean(baseDir)
	return &Store{
		BaseDir:     baseDir,
		PackagesDir: filepath.Join(baseDir, "packages"),
		ReleasesDir: filepath.Join(baseDir, "releases"),
	}
}

func (store *Store) EnsureDirs() error {
	directories := []string{store.BaseDir, store.PackagesDir, store.ReleasesDir}
	for _, directory := range directories {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			return fmt.Errorf("create dir failed: %s, %w", directory, err)
		}
	}
	return nil
}

func (store *Store) ReleaseDir(version string) string {
	version = sanitizeVersion(version)
	return filepath.Join(store.ReleasesDir, version)
}

func (store *Store) PackagePath(filename string) string {
	return filepath.Join(store.PackagesDir, filepath.Base(filename))
}

func (store *Store) StagePath(filename string) string {
	baseName := filepath.Base(filename)
	if baseName == "." || baseName == "/" || strings.TrimSpace(baseName) == "" {
		baseName = "package"
	}
	return filepath.Join(store.PackagesDir, fmt.Sprintf(".stage_%s", baseName))
}

func (store *Store) CurrentLinkPath() string {
	return filepath.Join(store.BaseDir, CurrentLinkName)
}

func (store *Store) LastGoodLinkPath() string {
	return filepath.Join(store.BaseDir, LastGoodLinkName)
}

func (store *Store) LinkTarget(linkPath string) (string, error) {
	targetPath, err := os.Readlink(linkPath)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(targetPath) {
		return filepath.Clean(targetPath), nil
	}
	return filepath.Clean(filepath.Join(filepath.Dir(linkPath), targetPath)), nil
}

func (store *Store) SetLink(linkPath, targetPath string) error {
	cleanLinkPath := filepath.Clean(linkPath)
	cleanTargetPath := filepath.Clean(targetPath)

	if err := os.MkdirAll(filepath.Dir(cleanLinkPath), 0o755); err != nil {
		return fmt.Errorf("prepare link dir failed: %w", err)
	}

	tempLink := cleanLinkPath + ".tmp"
	_ = os.Remove(tempLink)
	if err := os.Symlink(cleanTargetPath, tempLink); err != nil {
		return fmt.Errorf("create temp symlink failed: %w", err)
	}
	if err := os.Rename(tempLink, cleanLinkPath); err != nil {
		_ = os.Remove(tempLink)
		return fmt.Errorf("replace symlink failed: %w", err)
	}
	return nil
}

func sanitizeVersion(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		version = "unknown"
	}

	replacer := strings.NewReplacer("/", "_", "\\", "_", "..", "_", " ", "_")
	return replacer.Replace(version)
}
