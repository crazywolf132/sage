package update

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/crazywolf132/sage/internal/ui"
)

const lastCommitURL = "https://api.github.com/repos/sage/commits/main"

type commitResp struct {
	SHA string `json:"sha"`
}

func CheckForUpdates() error {
	checkFile, err := updateCheckPath()
	if err != nil {
		return nil
	}
	should, oldSHA := shouldCheck(checkFile)
	if !should {
		return nil
	}
	newSHA, err := getLatestSHA()
	if err != nil {
		return nil // ignore
	}
	_ = writeCheck(checkFile, newSHA)
	if oldSHA != "" && newSHA != oldSHA {
		ui.Warnf("A new version of Sage may be available!\n")
	}
	return nil
}

func updateCheckPath() (string, error) {
	if runtime.GOOS == "windows" {
		appdata := os.Getenv("LOCALAPPDATA")
		if appdata == "" {
			return "", fmt.Errorf("no LOCALAPPDATA")
		}
		return filepath.Join(appdata, "sage_update.json"), nil
	}
	return "/tmp/sage_update.json", nil
}

func shouldCheck(path string) (bool, string) {
	b, err := os.ReadFile(path)
	if err != nil {
		return true, ""
	}
	var st struct {
		LastCheck time.Time
		SHA       string
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return true, ""
	}
	if time.Since(st.LastCheck) < 24*time.Hour {
		return false, st.SHA
	}
	return true, st.SHA
}

func getLatestSHA() (string, error) {
	resp, err := http.Get(lastCommitURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	d, _ := io.ReadAll(resp.Body)
	var c commitResp
	if err := json.Unmarshal(d, &c); err != nil {
		return "", err
	}
	return c.SHA, nil
}

func writeCheck(path, sha string) error {
	st := struct {
		LastCheck time.Time
		SHA       string
	}{
		LastCheck: time.Now(),
		SHA:       sha,
	}
	b, err := json.Marshal(st)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0644)
}
