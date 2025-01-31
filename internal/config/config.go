package config

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/crazywolf132/sage/internal/git"
)

var (
	globalData = map[string]string{}
	localData  = map[string]string{}
)

// LoadAllConfigs reads the global + local config and merges them.
func LoadAllConfigs() error {
	if err := loadGlobalConfig(); err != nil {
		return err
	}
	// if inside a repo, load local
	g := git.NewShellGit()
	repo, err := g.IsRepo()
	if err == nil && repo {
		_ = loadLocalConfig() // if fails, ignore
	}
	return nil
}

func Get(key string) string {
	if val, ok := localData[key]; ok {
		return val
	}
	if val, ok := globalData[key]; ok {
		return val
	}
	return ""
}

func Set(key, value string) error {
	// if in repo, write local. else global
	g := git.NewShellGit()
	repo, err := g.IsRepo()
	if err == nil && repo {
		localData[key] = value
		return writeLocalConfig()
	}
	globalData[key] = value
	return writeGlobalConfig()
}

// load / write global

func globalPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".sage.toml"), nil
}

func loadGlobalConfig() error {
	p, err := globalPath()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	tmp := map[string]string{}
	if err := toml.Unmarshal(b, &tmp); err != nil {
		return err
	}
	globalData = tmp
	return nil
}

func writeGlobalConfig() error {
	p, err := globalPath()
	if err != nil {
		return err
	}
	b, err := toml.Marshal(globalData)
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0644)
}

// local

func localPath() string {
	return ".sage/config.toml"
}

func loadLocalConfig() error {
	b, err := os.ReadFile(localPath())
	if err != nil {
		return err
	}
	tmp := map[string]string{}
	if err := toml.Unmarshal(b, &tmp); err != nil {
		return err
	}
	localData = tmp
	return nil
}

func writeLocalConfig() error {
	os.MkdirAll(".sage", 0755)
	b, err := toml.Marshal(localData)
	if err != nil {
		return err
	}
	return os.WriteFile(localPath(), b, 0644)
}
