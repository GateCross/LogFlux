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
		return fmt.Errorf("activator store is nil")
	}
	if activator.CaddyLoader == nil {
		return fmt.Errorf("caddy loader is nil")
	}

	lock := activator.Lock
	if lock == nil {
		lock = &defaultActivateLock
	}
	lock.Lock()
	defer lock.Unlock()

	releaseDir := activator.Store.ReleaseDir(version)
	if !dirExists(releaseDir) {
		return fmt.Errorf("release dir not found: %s", releaseDir)
	}

	currentLink := activator.Store.CurrentLinkPath()
	lastGoodLink := activator.Store.LastGoodLinkPath()

	previousCurrentTarget, _ := activator.Store.LinkTarget(currentLink)
	if previousCurrentTarget != "" {
		if err := activator.Store.SetLink(lastGoodLink, previousCurrentTarget); err != nil {
			return fmt.Errorf("set last_good link failed: %w", err)
		}
	}

	if err := activator.Store.SetLink(currentLink, releaseDir); err != nil {
		return fmt.Errorf("set current link failed: %w", err)
	}

	if err := activator.CaddyLoader.Adapt(caddyConfig); err != nil {
		rollbackErr := rollbackToPrevious(activator.Store, previousCurrentTarget, caddyConfig, activator.CaddyLoader)
		if rollbackErr != nil {
			return fmt.Errorf("adapt failed: %v, rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("adapt failed: %w", err)
	}

	if err := activator.CaddyLoader.Load(caddyConfig); err != nil {
		rollbackErr := rollbackToPrevious(activator.Store, previousCurrentTarget, caddyConfig, activator.CaddyLoader)
		if rollbackErr != nil {
			return fmt.Errorf("load failed: %v, rollback failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("load failed: %w", err)
	}

	return nil
}

func rollbackToPrevious(store *Store, previousCurrentTarget, caddyConfig string, loader CaddyLoader) error {
	if strings.TrimSpace(previousCurrentTarget) == "" {
		return fmt.Errorf("no previous current target for rollback")
	}

	if err := store.SetLink(store.CurrentLinkPath(), previousCurrentTarget); err != nil {
		return err
	}

	if err := loader.Load(caddyConfig); err != nil {
		return fmt.Errorf("reload previous config failed: %w", err)
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
