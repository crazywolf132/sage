package git_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/crazywolf132/sage/internal/git"
)

func TestShellGit_Integration(t *testing.T) {
	tmp, err := os.MkdirTemp("", "sage-git-test-")
	assert.NoError(t, err)
	defer os.RemoveAll(tmp)

	err = os.Chdir(tmp)
	assert.NoError(t, err)

	_, _ = runCmd("git", "init") // ignoring output
	svc := git.NewShellGit()

	isRepo, err := svc.IsRepo()
	assert.NoError(t, err)
	assert.True(t, isRepo)

	// create a file
	err = os.WriteFile(filepath.Join(tmp, "test.txt"), []byte("Hello"), 0644)
	assert.NoError(t, err)

	clean, err := svc.IsClean()
	assert.NoError(t, err)
	assert.False(t, clean)

	// stage
	err = svc.StageAll()
	assert.NoError(t, err)

	// commit
	err = svc.Commit("Initial commit", false)
	assert.NoError(t, err)

	clean, err = svc.IsClean()
	assert.NoError(t, err)
	assert.True(t, clean)
}

func runCmd(prog string, args ...string) (string, error) {
	cmd := exec.Command(prog, args...)
	b, err := cmd.CombinedOutput()
	return string(b), err
}
