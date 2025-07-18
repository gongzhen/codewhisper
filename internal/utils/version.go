package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	// CurrentVersion is the current version of CodeWhisper
	CurrentVersion = "0.1.0"

	// GitHub repo in format owner/repo
	githubRepo = "gongzhen/codewhisper-go" // ← 👈 记得改成你自己的 repo 路径
)

// GitHubRelease represents GitHub API response
type GitHubRelease struct {
	TagName string `json:"tag_name"`
}

// CheckForUpdates checks GitHub for the latest release
func CheckForUpdates() (bool, string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", githubRepo)

	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return false, "", err
	}

	latest := release.TagName
	if latest != "" && latest != CurrentVersion {
		return true, latest, nil
	}

	return false, latest, nil
}
