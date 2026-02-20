package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	updateRepoOwner = "lemon5227"
	updateRepoName  = "aegis"
)

type UpdateStatus struct {
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	HasUpdate      bool   `json:"hasUpdate"`
	ReleaseURL     string `json:"releaseURL"`
	ReleaseNotes   string `json:"releaseNotes"`
	PublishedAt    int64  `json:"publishedAt"`
	CheckedAt      int64  `json:"checkedAt"`
	ErrorMessage   string `json:"errorMessage"`
}

type VersionHistoryItem struct {
	Version     string `json:"version"`
	PublishedAt int64  `json:"publishedAt"`
	Summary     string `json:"summary"`
	URL         string `json:"url"`
}

type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
}

func currentAppVersion() string {
	v := strings.TrimSpace(os.Getenv("AEGIS_APP_VERSION"))
	if v == "" {
		v = "v0.1.0-dev"
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

func (a *App) CheckForUpdates() (UpdateStatus, error) {
	now := time.Now().Unix()
	status := UpdateStatus{
		CurrentVersion: currentAppVersion(),
		CheckedAt:      now,
	}

	releases, err := fetchGitHubReleases(8)
	if err != nil {
		status.ErrorMessage = "Unable to reach update source"
		return status, nil
	}
	if len(releases) == 0 {
		status.ErrorMessage = "No release information available"
		return status, nil
	}

	latest := releases[0]
	status.LatestVersion = normalizeVersionTag(latest.TagName)
	status.ReleaseURL = strings.TrimSpace(latest.HTMLURL)
	status.ReleaseNotes = strings.TrimSpace(latest.Body)
	status.PublishedAt = parseGitHubTime(latest.PublishedAt)

	cmp := compareSemver(status.LatestVersion, status.CurrentVersion)
	status.HasUpdate = cmp > 0
	return status, nil
}

func (a *App) GetVersionHistory(limit int) ([]VersionHistoryItem, error) {
	if limit <= 0 || limit > 20 {
		limit = 8
	}
	releases, err := fetchGitHubReleases(limit)
	if err != nil {
		return []VersionHistoryItem{}, nil
	}

	result := make([]VersionHistoryItem, 0, len(releases))
	for _, item := range releases {
		summary := strings.TrimSpace(item.Name)
		if summary == "" {
			summary = firstNonEmptyLine(item.Body)
		}
		if summary == "" {
			summary = "Release " + normalizeVersionTag(item.TagName)
		}
		if len([]rune(summary)) > 140 {
			r := []rune(summary)
			summary = string(r[:140]) + "..."
		}

		result = append(result, VersionHistoryItem{
			Version:     normalizeVersionTag(item.TagName),
			PublishedAt: parseGitHubTime(item.PublishedAt),
			Summary:     summary,
			URL:         strings.TrimSpace(item.HTMLURL),
		})
	}

	return result, nil
}

func fetchGitHubReleases(limit int) ([]githubRelease, error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d", updateRepoOwner, updateRepoName, limit)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "aegis-update-checker")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("github releases status: %d", resp.StatusCode)
	}

	var payload []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	result := make([]githubRelease, 0, len(payload))
	for _, item := range payload {
		if item.Draft {
			continue
		}
		result = append(result, item)
	}
	return result, nil
}

func normalizeVersionTag(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "v0.0.0"
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

func compareSemver(a string, b string) int {
	av := parseSemverCore(a)
	bv := parseSemverCore(b)
	for i := 0; i < 3; i++ {
		if av[i] > bv[i] {
			return 1
		}
		if av[i] < bv[i] {
			return -1
		}
	}
	return 0
}

func parseSemverCore(v string) [3]int {
	v = strings.TrimSpace(strings.TrimPrefix(v, "v"))
	if idx := strings.IndexAny(v, "-+"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	result := [3]int{0, 0, 0}
	for i := 0; i < len(parts) && i < 3; i++ {
		n, err := strconv.Atoi(strings.TrimSpace(parts[i]))
		if err != nil {
			continue
		}
		result[i] = n
	}
	return result
}

func parseGitHubTime(v string) int64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	t, err := time.Parse(time.RFC3339, v)
	if err != nil {
		return 0
	}
	return t.Unix()
}

func firstNonEmptyLine(body string) string {
	for _, line := range strings.Split(body, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}
