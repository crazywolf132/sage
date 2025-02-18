package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
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
	"golang.org/x/crypto/pbkdf2"
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
		// Check if this is a sensitive key that needs decryption
		for _, k := range sensitiveKeys {
			if strings.HasPrefix(strings.ToLower(key), strings.ToLower(k)) {
				decrypted, err := decryptValue(val)
				if err != nil {
					ui.Warnf("Failed to decrypt sensitive value: %v\n", err)
					return ""
				}
				return decrypted
			}
		}
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

// Unset removes a configuration value
func Unset(key string, global bool) error {
	if global {
		delete(globalData, key)
		return writeGlobalConfig()
	}
	delete(localData, key)
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
	g := git.NewShellGit()
	gitDir, err := g.Run("rev-parse", "--git-dir")
	if err != nil {
		return ""
	}
	return filepath.Join(strings.TrimSpace(gitDir), ".sage/config.toml")
}

func loadLocalConfig() error {
	path := localPath()
	if path == "" {
		return fmt.Errorf("not in a git repository")
	}
	b, err := os.ReadFile(path)
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
	path := localPath()
	if path == "" {
		return fmt.Errorf("not in a git repository")
	}

	// Ensure the .git/.sage folder exists.
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

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
	return os.WriteFile(path, b, 0644)
}

// getMasterKey derives a master encryption key from system-specific data
func getMasterKey() ([]byte, error) {
	// Get system-specific data to derive the key from
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Get machine ID if available (usually in /etc/machine-id or /var/lib/dbus/machine-id)
	var machineID string
	if runtime.GOOS != "windows" {
		if id, err := os.ReadFile("/etc/machine-id"); err == nil {
			machineID = string(id)
		} else if id, err := os.ReadFile("/var/lib/dbus/machine-id"); err == nil {
			machineID = string(id)
		}
	}

	// Combine system data into a unique string
	systemData := fmt.Sprintf("%s:%s:%s:%s", homeDir, runtime.GOOS, runtime.GOARCH, machineID)

	// Use PBKDF2 to derive a secure key
	salt := []byte("sage-config-v1") // Version this so we can change it if needed
	return pbkdf2.Key([]byte(systemData), salt, 100000, 32, sha256.New), nil
}

// encryptValue encrypts sensitive configuration values using AES-GCM
func encryptValue(value string) (string, error) {
	masterKey, err := getMasterKey()
	if err != nil {
		return "", fmt.Errorf("failed to get master key: %w", err)
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate a random nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt and authenticate the value
	ciphertext := gcm.Seal(nonce, nonce, []byte(value), nil)

	// Encode the result
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryptValue decrypts sensitive configuration values
func decryptValue(encrypted string) (string, error) {
	masterKey, err := getMasterKey()
	if err != nil {
		return "", fmt.Errorf("failed to get master key: %w", err)
	}

	// Decode the base64 data
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted value: %w", err)
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(data) < gcm.NonceSize() {
		return "", fmt.Errorf("invalid ciphertext length")
	}

	nonce := data[:gcm.NonceSize()]
	ciphertext := data[gcm.NonceSize():]

	// Decrypt and verify the value
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt value (possibly corrupted or from different system)")
	}

	return string(plaintext), nil
}

// IsExperimentalFeatureEnabled checks if an experimental feature is enabled.
// It first checks the local config, then falls back to global config.
// The feature can be enabled by setting experimental.<feature_name>=true
func IsExperimentalFeatureEnabled(featureName string) bool {
	configKey := fmt.Sprintf("experimental.%s", featureName)

	// Check local config first
	g := git.NewShellGit()
	repo, err := g.IsRepo()
	if err == nil && repo {
		if val, ok := localData[configKey]; ok {
			return strings.ToLower(val) == "true"
		}
	}

	// Fall back to global config
	if val, ok := globalData[configKey]; ok {
		return strings.ToLower(val) == "true"
	}

	return false // Default to disabled
}
