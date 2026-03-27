package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultGitHubAPIBaseURL = "https://api.github.com"

type githubReleaseResponse struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type releaseInfo struct {
	Version      string
	TagName      string
	ReleaseURL   string
	ReleaseNotes string
	Assets       map[string]string
}

func (m *Manager) fetchLatestRelease(ctx context.Context) (*releaseInfo, error) {
	if strings.TrimSpace(m.repo) == "" {
		return nil, fmt.Errorf("update repository is not configured")
	}

	apiBase := strings.TrimRight(strings.TrimSpace(m.githubAPIBaseURL), "/")
	if apiBase == "" {
		apiBase = defaultGitHubAPIBaseURL
	}

	url := fmt.Sprintf("%s/repos/%s/releases/latest", apiBase, m.repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "spoutmc-updater")
	if m.githubToken != "" {
		req.Header.Set("Authorization", "Bearer "+m.githubToken)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("github releases API returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var payload githubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decode latest release response: %w", err)
	}
	if strings.TrimSpace(payload.TagName) == "" {
		return nil, fmt.Errorf("latest release response missing tag_name")
	}

	assets := make(map[string]string, len(payload.Assets))
	for _, asset := range payload.Assets {
		if asset.Name == "" || asset.BrowserDownloadURL == "" {
			continue
		}
		assets[asset.Name] = asset.BrowserDownloadURL
	}

	return &releaseInfo{
		Version:      normalizeVersion(payload.TagName),
		TagName:      payload.TagName,
		ReleaseURL:   payload.HTMLURL,
		ReleaseNotes: payload.Body,
		Assets:       assets,
	}, nil
}

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 60 * time.Second}
}
