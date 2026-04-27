package waf

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type CaddyLoader interface {
	Adapt(config string) error
	Load(config string) error
}

type Activator struct {
	Store       *Store
	CaddyLoader CaddyLoader
	Lock        *sync.Mutex
}

var defaultActivateLock sync.Mutex

func (activator *Activator) ActivateVersion(version, caddyConfig string) error {
	if activator == nil || activator.Store == nil {
		return fmt.Errorf("激活器存储为空")
	}
	if activator.CaddyLoader == nil {
		return fmt.Errorf("Caddy 加载器为空")
	}

	lock := activator.Lock
	if lock == nil {
		lock = &defaultActivateLock
	}
	lock.Lock()
	defer lock.Unlock()

	releaseDir := activator.Store.ReleaseDir(version)
	if !dirExists(releaseDir) {
		return fmt.Errorf("发布目录不存在: %s", releaseDir)
	}

	currentLink := activator.Store.CurrentLinkPath()
	lastGoodLink := activator.Store.LastGoodLinkPath()

	previousCurrentTarget, _ := activator.Store.LinkTarget(currentLink)
	if previousCurrentTarget != "" {
		if err := activator.Store.SetLink(lastGoodLink, previousCurrentTarget); err != nil {
			return fmt.Errorf("设置 last_good 链接失败: %w", err)
		}
	}

	if err := activator.Store.SetLink(currentLink, releaseDir); err != nil {
		return fmt.Errorf("设置 current 链接失败: %w", err)
	}

	if err := activator.CaddyLoader.Adapt(caddyConfig); err != nil {
		rollbackErr := rollbackToPrevious(activator.Store, previousCurrentTarget, caddyConfig, activator.CaddyLoader)
		if rollbackErr != nil {
			return fmt.Errorf("适配失败: %v，回滚失败: %v", err, rollbackErr)
		}
		return fmt.Errorf("适配失败: %w", err)
	}

	if err := activator.CaddyLoader.Load(caddyConfig); err != nil {
		rollbackErr := rollbackToPrevious(activator.Store, previousCurrentTarget, caddyConfig, activator.CaddyLoader)
		if rollbackErr != nil {
			return fmt.Errorf("加载失败: %v，回滚失败: %v", err, rollbackErr)
		}
		return fmt.Errorf("加载失败: %w", err)
	}

	return nil
}

func rollbackToPrevious(store *Store, previousCurrentTarget, caddyConfig string, loader CaddyLoader) error {
	if strings.TrimSpace(previousCurrentTarget) == "" {
		return fmt.Errorf("没有可回滚的上一版 current 目标")
	}

	if err := store.SetLink(store.CurrentLinkPath(), previousCurrentTarget); err != nil {
		return err
	}

	if err := loader.Load(caddyConfig); err != nil {
		return fmt.Errorf("重载上一版配置失败: %w", err)
	}

	return nil
}

func dirExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	stat, err := os.Stat(filepath.Clean(path))
	if err != nil {
		return false
	}
	return stat.IsDir()
}
