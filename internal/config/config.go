package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"

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

func Get(key string, useLocal bool) string {
	// If useLocal is true and we're in a repo, check local first
	if useLocal {
		g := git.NewShellGit()
		repo, err := g.IsRepo()
		if err == nil && repo {
			if val, ok := localData[key]; ok {
				return val
			}
		}
	}
	// Otherwise check global
	if val, ok := globalData[key]; ok {
		return val
	}
	return ""
}

func Set(key, value string, useLocal bool) error {
	if useLocal {
		g := git.NewShellGit()
		repo, err := g.IsRepo()
		if err != nil {
			return err
		}
		if !repo {
			return errors.New("not in a git repository")
		}
		localData[key] = value
		return writeLocalConfig()
	}
	globalData[key] = value
	return writeGlobalConfig()
}

// load / write global

func globalPath() (string, error) {
	var configDir string

	switch runtime.GOOS {
	case "windows":
		// On Windows, use %APPDATA%
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		configDir = filepath.Join(appData, "sage")
	case "darwin":
		// On macOS, use ~/Library/Application Support
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(home, "Library", "Application Support", "sage")
	default:
		// On Linux and others, use XDG_CONFIG_HOME or ~/.config
		xdgConfig := os.Getenv("XDG_CONFIG_HOME")
		if xdgConfig == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			xdgConfig = filepath.Join(home, ".config")
		}
		configDir = filepath.Join(xdgConfig, "sage")
	}

	// Ensure the config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(configDir, "config.toml"), nil
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
