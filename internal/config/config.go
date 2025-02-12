package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/crazywolf132/sage/internal/git"
	"github.com/crazywolf132/sage/internal/ui"
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

func Set(key, value string, global bool) error {
	if !global {
		// Check if this is a sensitive key that shouldn't be stored locally
		for _, k := range sensitiveKeys {
			if strings.HasPrefix(strings.ToLower(key), strings.ToLower(k)) {
				return fmt.Errorf("security: sensitive keys like %s must be set globally", key)
			}
		}
	}

	// For sensitive keys in global config, encrypt the value
	if global {
		for _, k := range sensitiveKeys {
			if strings.HasPrefix(strings.ToLower(key), strings.ToLower(k)) {
				encryptedValue, err := encryptValue(value)
				if err != nil {
					return fmt.Errorf("failed to secure sensitive value: %w", err)
				}
				value = encryptedValue
				break
			}
		}
	}

	if global {
		globalData[key] = value
		return writeGlobalConfig()
	}
	localData[key] = value
	return writeLocalConfig()
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

// KEYS WE DON'T WANT TO SAVE LOCALLY:
var sensitiveKeys = []string{
	"api.api_key",
	"github.token",
	"openai.api_key",
	"ai.api_key",
	"auth.token",
	"credentials.token",
}

func writeLocalConfig() error {
	// Ensure the .sage folder exists.
	os.MkdirAll(".sage", 0755)

	// Remove sensitive keys before saving.
	for _, naughtyKey := range sensitiveKeys {
		// We will inform the user that we are not saving this key. Please add it to global config instead.
		if _, ok := localData[naughtyKey]; ok {
			delete(localData, naughtyKey)
			fmt.Println(ui.Gray(fmt.Sprintf("Warning: not saving sensitive key %s in local config", naughtyKey)))
		}
	}

	b, err := toml.Marshal(localData)
	if err != nil {
		return err
	}
	return os.WriteFile(localPath(), b, 0644)
}

// encryptValue encrypts sensitive configuration values
func encryptValue(value string) (string, error) {
	key := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate encryption key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Store key alongside encrypted value
	ciphertext := gcm.Seal(nonce, nonce, []byte(value), nil)
	return base64.StdEncoding.EncodeToString(append(key, ciphertext...)), nil
}

// decryptValue decrypts sensitive configuration values
func decryptValue(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted value: %w", err)
	}

	if len(data) < 32 {
		return "", fmt.Errorf("invalid encrypted value format")
	}

	key := data[:32]
	ciphertext := data[32:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid ciphertext length")
	}

	nonce := ciphertext[:gcm.NonceSize()]
	ciphertext = ciphertext[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt value: %w", err)
	}

	return string(plaintext), nil
}
