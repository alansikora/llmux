package update

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/allskar/llmux/internal/config"
)

const (
	repo          = "alansikora/llmux"
	cacheTTL      = 24 * time.Hour
	cacheFile     = "update-check.json"
	apiTimeout    = 5 * time.Second
	fetchTimeout  = 60 * time.Second
	maxBinarySize = 100 << 20 // 100 MB
)

// allowedDownloadHosts are the only hosts we'll fetch release assets from.
var allowedDownloadHosts = []string{
	"github.com",
	"objects.githubusercontent.com",
}

// Release holds information about a GitHub release.
type Release struct {
	TagName string `json:"tag_name"`
}

// cachedCheck stores the last update check result.
type cachedCheck struct {
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
}

func cachePath() string {
	return filepath.Join(config.ConfigDir(), cacheFile)
}

// readCache returns the cached check result, or nil if expired/missing.
func readCache() *cachedCheck {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return nil
	}
	var c cachedCheck
	if err := json.Unmarshal(data, &c); err != nil {
		return nil
	}
	if time.Since(c.CheckedAt) > cacheTTL {
		return nil
	}
	return &c
}

func writeCache(version string) {
	c := cachedCheck{
		LatestVersion: version,
		CheckedAt:     time.Now(),
	}
	data, _ := json.Marshal(c)
	os.MkdirAll(filepath.Dir(cachePath()), 0755)
	os.WriteFile(cachePath(), data, 0644)
}

// CheckLatest returns the latest release version from GitHub.
// Uses a file cache to avoid hitting the API on every invocation.
func CheckLatest() (string, error) {
	if c := readCache(); c != nil {
		return c.LatestVersion, nil
	}

	client := &http.Client{Timeout: apiTimeout}
	resp, err := client.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("github api: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}

	writeCache(release.TagName)
	return release.TagName, nil
}

// IsNewer returns true if latest is a newer version than current.
// Both should be in "vX.Y.Z" format.
func IsNewer(current, latest string) bool {
	cur := parseVersion(current)
	lat := parseVersion(latest)
	if cur == nil || lat == nil {
		return false
	}
	for i := 0; i < 3; i++ {
		if lat[i] > cur[i] {
			return true
		}
		if lat[i] < cur[i] {
			return false
		}
	}
	return false
}

func parseVersion(v string) []int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return nil
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n := 0
		for _, c := range p {
			if c < '0' || c > '9' {
				return nil
			}
			n = n*10 + int(c-'0')
		}
		nums[i] = n
	}
	return nums
}

// CheckUpdateNoticeCached returns a notice string if an update is available
// using only the local cache. Returns empty string on cache miss (never hits network).
func CheckUpdateNoticeCached(currentVersion string) string {
	c := readCache()
	if c == nil {
		return ""
	}
	if IsNewer(currentVersion, c.LatestVersion) {
		return c.LatestVersion
	}
	return ""
}

// CheckUpdateNoticeAsync starts a background update check and sends the result
// on the returned channel. The caller can select on the channel with a short
// timeout to avoid blocking. On cache hit the result is available immediately.
func CheckUpdateNoticeAsync(currentVersion string) <-chan string {
	ch := make(chan string, 1)
	// Fast path: cache hit
	if notice := CheckUpdateNoticeCached(currentVersion); notice != "" {
		ch <- notice
		return ch
	}
	go func() {
		latest, err := CheckLatest()
		if err != nil {
			ch <- ""
			return
		}
		if IsNewer(currentVersion, latest) {
			ch <- latest
		} else {
			ch <- ""
		}
	}()
	return ch
}

// Upgrade downloads and installs the latest (or specified) version.
func Upgrade(targetVersion string) error {
	fetchedLatest := false
	if targetVersion == "" {
		latest, err := FetchLatest()
		if err != nil {
			return fmt.Errorf("fetching latest version: %w", err)
		}
		targetVersion = latest
		fetchedLatest = true
	}

	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Fetch the release asset list once
	release, err := fetchReleaseAssets(targetVersion)
	if err != nil {
		return err
	}

	assetURL, err := findAssetURLFrom(release, targetVersion, osName, arch)
	if err != nil {
		return err
	}

	checksumURL, err := findChecksumURLFrom(release, targetVersion)
	if err != nil {
		return err
	}

	// Download the checksums file
	expectedHash, err := fetchExpectedChecksum(checksumURL, osName, arch)
	if err != nil {
		return fmt.Errorf("fetching checksums: %w", err)
	}

	// Download the archive
	client := &http.Client{Timeout: fetchTimeout}
	resp, err := client.Get(assetURL)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	// Extract the binary to a temp file
	tmpDir, err := os.MkdirTemp("", "llmux-upgrade-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	// Save the archive to disk so we can checksum it
	archivePath := filepath.Join(tmpDir, "archive.tar.gz")
	archiveFile, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	hasher := sha256.New()
	if _, err := io.Copy(archiveFile, io.TeeReader(io.LimitReader(resp.Body, maxBinarySize), hasher)); err != nil {
		archiveFile.Close()
		return fmt.Errorf("downloading archive: %w", err)
	}
	archiveFile.Close()

	// Verify checksum
	actualHash := hex.EncodeToString(hasher.Sum(nil))
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	// Re-open and extract
	archiveReader, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer archiveReader.Close()

	tmpBin := filepath.Join(tmpDir, "llmux")
	if err := extractBinary(archiveReader, tmpBin); err != nil {
		return fmt.Errorf("extracting: %w", err)
	}

	// Find current binary path
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolving binary path: %w", err)
	}

	// Replace current binary
	if err := replaceBinary(tmpBin, exe); err != nil {
		return err
	}

	// Refresh the cache only when targetVersion came from FetchLatest — an
	// explicit targetVersion could be a downgrade, and recording it as the
	// latest would suppress valid update notices for up to the cache TTL.
	if fetchedLatest {
		writeCache(targetVersion)
	}

	return nil
}

// FetchLatest gets the latest release tag directly from GitHub, bypassing
// the cache. It refreshes the cache on success so ambient notices stay in
// sync. Use this for explicit user actions like `llmux upgrade`; use
// CheckLatest for passive checks that should respect the TTL.
func FetchLatest() (string, error) {
	client := &http.Client{Timeout: apiTimeout}
	resp, err := client.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("github api: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", err
	}
	writeCache(release.TagName)
	return release.TagName, nil
}

type releaseAssets struct {
	Assets []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// fetchReleaseAssets fetches the asset list for a given release tag.
func fetchReleaseAssets(tag string) (*releaseAssets, error) {
	client := &http.Client{Timeout: apiTimeout}
	resp, err := client.Get(fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, tag))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("release %s not found: %s", tag, resp.Status)
	}

	var release releaseAssets
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

// validateAssetURL checks that a download URL points to an allowed GitHub host.
func validateAssetURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid asset URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("asset URL must use https, got %s", u.Scheme)
	}
	for _, allowed := range allowedDownloadHosts {
		if u.Host == allowed || strings.HasSuffix(u.Host, "."+allowed) {
			return nil
		}
	}
	return fmt.Errorf("asset URL host %q is not an allowed GitHub domain", u.Host)
}

// findAssetURLFrom finds the download URL for a specific release asset.
func findAssetURLFrom(release *releaseAssets, tag, osName, arch string) (string, error) {
	suffix := fmt.Sprintf("_%s_%s.tar.gz", osName, arch)
	for _, a := range release.Assets {
		if strings.HasSuffix(a.Name, suffix) {
			if err := validateAssetURL(a.BrowserDownloadURL); err != nil {
				return "", err
			}
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("no asset found for %s/%s in release %s", osName, arch, tag)
}

// findChecksumURLFrom finds the checksums.txt download URL for a release.
func findChecksumURLFrom(release *releaseAssets, tag string) (string, error) {
	for _, a := range release.Assets {
		if a.Name == "checksums.txt" {
			if err := validateAssetURL(a.BrowserDownloadURL); err != nil {
				return "", err
			}
			return a.BrowserDownloadURL, nil
		}
	}
	return "", fmt.Errorf("checksums.txt not found in release %s", tag)
}

// fetchExpectedChecksum downloads the checksums file and extracts the hash
// for the matching archive.
func fetchExpectedChecksum(checksumURL, osName, arch string) (string, error) {
	client := &http.Client{Timeout: apiTimeout}
	resp, err := client.Get(checksumURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("downloading checksums: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB max
	if err != nil {
		return "", err
	}

	suffix := fmt.Sprintf("_%s_%s.tar.gz", osName, arch)
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: "<hash>  <filename>"
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		if strings.HasSuffix(parts[1], suffix) {
			return parts[0], nil
		}
	}
	return "", fmt.Errorf("no checksum found for %s/%s", osName, arch)
}

// extractBinary extracts the llmux binary from a tar.gz stream.
func extractBinary(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if filepath.Base(hdr.Name) == "llmux" && hdr.Typeflag == tar.TypeReg {
			if hdr.Size > maxBinarySize {
				return fmt.Errorf("binary too large: %d bytes (max %d)", hdr.Size, maxBinarySize)
			}
			f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
			if err != nil {
				return err
			}
			defer f.Close()
			n, err := io.Copy(f, io.LimitReader(tr, maxBinarySize))
			if err != nil {
				return err
			}
			if hdr.Size > 0 && n < hdr.Size {
				return fmt.Errorf("binary truncated: wrote %d of %d bytes", n, hdr.Size)
			}
			return nil
		}
	}
	return fmt.Errorf("llmux binary not found in archive")
}

// replaceBinary atomically replaces the target binary with the new one.
func replaceBinary(src, dst string) error {
	// Copy to a temp file next to the destination (same filesystem for atomic rename)
	dir := filepath.Dir(dst)
	tmp, err := os.CreateTemp(dir, ".llmux-new-*")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpPath := tmp.Name()

	srcFile, err := os.Open(src)
	if err != nil {
		os.Remove(tmpPath)
		return err
	}
	defer srcFile.Close()

	if _, err := io.Copy(tmp, srcFile); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	tmp.Close()

	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("replacing binary: %w", err)
	}

	return nil
}
