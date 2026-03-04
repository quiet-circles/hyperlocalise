package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/spf13/cobra"
)

const (
	latestReleaseURL = "https://api.github.com/repos/quiet-circles/hyperlocalise/releases/latest"
	upgradeCommand   = "hyperlocalise update"
)

type releaseInfo struct {
	TagName string `json:"tag_name"`
}

var latestVersionFetcher = fetchLatestVersion

func notifyIfUpdateAvailable(cmd *cobra.Command, version string) {
	currentVersion, ok := normalizeSemver(version)
	if !ok {
		return
	}

	latestVersion, err := latestVersionFetcher(cmd.Context())
	if err != nil {
		return
	}

	if !latestVersion.GreaterThan(currentVersion) {
		return
	}

	_, _ = fmt.Fprintf(
		cmd.ErrOrStderr(),
		"\nUpdate available %s → %s\nCurrent version: %s\nLatest version: %s\nUpgrade: %s\n",
		currentVersion.Original(),
		latestVersion.Original(),
		currentVersion.Original(),
		latestVersion.Original(),
		upgradeCommand,
	)
}

func fetchLatestVersion(ctx context.Context) (*semver.Version, error) {
	requestCtx, cancel := context.WithTimeout(ctx, 1200*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create latest release request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request latest release: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			// Response body has already been consumed; close errors are non-fatal here.
			_ = closeErr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("latest release returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read latest release response: %w", err)
	}

	var release releaseInfo
	if err := json.Unmarshal(body, &release); err != nil {
		return nil, fmt.Errorf("decode latest release response: %w", err)
	}

	latestVersion, ok := normalizeSemver(release.TagName)
	if !ok {
		return nil, fmt.Errorf("latest release tag %q is not semver", release.TagName)
	}

	return latestVersion, nil
}

func normalizeSemver(version string) (*semver.Version, bool) {
	trimmed := strings.TrimSpace(version)
	if trimmed == "" {
		return nil, false
	}

	trimmed = strings.TrimPrefix(trimmed, "v")

	parsed, err := semver.NewVersion(trimmed)
	if err != nil {
		return nil, false
	}

	return parsed, true
}
