package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/crazywolf132/sage/internal/config"
)

func TestConfig_GlobalReadWrite(t *testing.T) {
	// set a temp HOME
	tmpHome := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", oldHome)

	// ensure no config file initially
	err := config.LoadAllConfigs()
	assert.NoError(t, err)

	assert.Equal(t, "", config.Get("testKey"))

	err = config.Set("testKey", "testValue")
	assert.NoError(t, err)

	val := config.Get("testKey")
	assert.Equal(t, "testValue", val)

	// check the file exists
	cfgPath := filepath.Join(tmpHome, ".sage.toml")
	_, statErr := os.Stat(cfgPath)
	assert.NoError(t, statErr)
}

// For local config test, you'd create a .git and .sage/config.toml,
// but we skip details here for brevity.
