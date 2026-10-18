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

	// LayoutBuilder 在激活前把发布目录重构成 Caddy 可直接加载的布局。
	//
	// 为空时退化为仅切换符号链接，用于测试或规则已就绪的场景。
	LayoutBuilder func(releaseDir, layoutRoot string) error
}

var defaultActivateLock sync.Mutex

// LayoutRoot 返回激活版本的布局根目录。
//
// 布局必须落在 current 链接外部：current 指向 release 目录，而布局是生成物，
// 若写进 release 目录会污染版本内容并阻碍回滚。
func (activator *Activator) LayoutRoot() string {
	if activator == nil || activator.Store == nil {
		return ""
	}
	return filepath.Join(activator.Store.BaseDir, "active")
}

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

	// 布局必须先于 Caddy 加载构建完成：Caddyfile 中的 Include 指向布局路径，
	// 若顺序颠倒，Adapt/Load 会因找不到规则文件而失败。
	if err := activator.buildLayout(version, releaseDir); err != nil {
		return err
	}

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

// buildLayout 在激活前把发布目录重构成 Caddy 可加载的布局。
func (activator *Activator) buildLayout(version, releaseDir string) error {
	layoutRoot := activator.LayoutRoot()
	if strings.TrimSpace(layoutRoot) == "" {
		return fmt.Errorf("布局目录为空")
	}

	// 先清理再重建，保证布局与当前版本严格一致，不残留上一个版本的文件。
	if err := os.RemoveAll(layoutRoot); err != nil {
		return fmt.Errorf("清理旧布局失败: %w", err)
	}

	if activator.LayoutBuilder != nil {
		return activator.LayoutBuilder(releaseDir, layoutRoot)
	}

	// 默认实现：直接复用发布目录内的规则树，不额外拷贝。
	return linkOrCopyCRSRules(releaseDir, layoutRoot)
}

// rollbackToPrevious 在激活失败时把 current 指回上一个版本并重载配置。
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
