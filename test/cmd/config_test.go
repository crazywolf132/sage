package cmd_test

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestConfigSetAndGet(t *testing.T) {
	// Setup temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".sage.yaml")
	viper.SetConfigFile(configFile)

	// Test setting a value
	key := "defaultBranch"
	value := "develop"

	// Set the value
	viper.Set(key, value)
	err := viper.WriteConfig()
	assert.NoError(t, err)

	// Get the value back
	result := viper.GetString(key)
	assert.Equal(t, value, result)

	// Test non-existent key
	nonExistentKey := "nonexistent"
	result = viper.GetString(nonExistentKey)
	assert.Empty(t, result)
}

func TestConfigShow(t *testing.T) {
	// Setup temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, ".sage.yaml")
	viper.SetConfigFile(configFile)

	// Set multiple values
	viper.Set("defaultBranch", "main")
	viper.Set("explain", true)
	err := viper.WriteConfig()
	assert.NoError(t, err)

	// Get all settings
	settings := viper.AllSettings()
	assert.NotEmpty(t, settings)
	assert.Equal(t, "main", settings["defaultbranch"])
	assert.Equal(t, true, settings["explain"])
}
