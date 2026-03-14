// Package selfupdate updates the atb binary from GitHub Releases.
package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	// DefaultRepo is the GitHub repository used for self-updates.
	DefaultRepo = "ametel01/agents-toolbelt"

	binaryName           = "atb"
	checksumAssetName    = "checksums.txt"
	maxReleaseBinarySize = 64 << 20
)

var (
	errAssetNotFound      = errors.New("release asset not found for platform")
	errChecksumMismatch   = errors.New("checksum verification failed")
	errChecksumNotFound   = errors.New("checksum not found for asset")
	errInvalidVersion     = errors.New("invalid version")
	errInvalidBinarySize  = errors.New("release binary has invalid size")
	errBinaryTooLarge     = errors.New("release binary exceeds size limit")
	errUnsupportedArch    = errors.New("unsupported architecture")
	errUnsupportedOS      = errors.New("unsupported operating system")
	errUnexpectedResponse = errors.New("unexpected release response")
)

// Options configures a self-update attempt.
type Options struct {
	BaseURL        string
	Client         *http.Client
	CurrentVersion string
	ExecutablePath string
	GOARCH         string
	GOOS           string
	Repo           string
}

// Result describes the outcome of a self-update attempt.
type Result struct {
	CurrentVersion string
	ExecutablePath string
	LatestVersion  string
	Updated        bool
}

type githubRelease struct {
	Assets  []githubAsset `json:"assets"`
	TagName string        `json:"tag_name"`
}

type githubAsset struct {
	BrowserDownloadURL string `json:"browser_download_url"`
	Name               string `json:"name"`
}

// Update checks GitHub Releases for a newer atb version and replaces the current binary when needed.
func Update(ctx context.Context, opts Options) (Result, error) {
	currentVersion, err := normalizeVersion(opts.CurrentVersion)
	if err != nil {
		return Result{}, fmt.Errorf("normalize current version: %w", err)
	}

	goos, err := normalizeOS(opts.GOOS)
	if err != nil {
		return Result{}, err
	}

	goarch, err := normalizeArch(opts.GOARCH)
	if err != nil {
		return Result{}, err
	}

	executablePath, err := resolveExecutablePath(opts.ExecutablePath)
	if err != nil {
		return Result{}, fmt.Errorf("resolve executable path: %w", err)
	}

	repo := opts.Repo
	if repo == "" {
		repo = DefaultRepo
	}

	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	release, err := latestRelease(ctx, client, repo, opts.BaseURL)
	if err != nil {
		return Result{}, err
	}

	latestVersion, err := normalizeVersion(release.TagName)
	if err != nil {
		return Result{}, fmt.Errorf("normalize latest version: %w", err)
	}

	result := Result{
		CurrentVersion: currentVersion,
		ExecutablePath: executablePath,
		LatestVersion:  latestVersion,
	}

	if semver.Compare(currentVersion, latestVersion) >= 0 {
		return result, nil
	}

	if err := fetchVerifyAndReplace(ctx, client, release, goos, goarch, executablePath); err != nil {
		return Result{}, err
	}

	result.Updated = true

	return result, nil
}

func normalizeVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	if version != "" && version[0] != 'v' {
		version = "v" + version
	}

	if !semver.IsValid(version) {
		return "", fmt.Errorf("%w: %q", errInvalidVersion, version)
	}

	return semver.Canonical(version), nil
}

func normalizeOS(goos string) (string, error) {
	switch goos {
	case "linux", "darwin":
		return goos, nil
	default:
		return "", fmt.Errorf("%w: %q", errUnsupportedOS, goos)
	}
}

func normalizeArch(goarch string) (string, error) {
	switch goarch {
	case "amd64", "arm64":
		return goarch, nil
	default:
		return "", fmt.Errorf("%w: %q", errUnsupportedArch, goarch)
	}
}

func resolveExecutablePath(path string) (string, error) {
	if path == "" {
		var err error
		path, err = os.Executable()
		if err != nil {
			return "", fmt.Errorf("resolve current executable path: %w", err)
		}
	}

	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}

	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("resolve executable symlinks for %s: %w", path, err)
	}

	return path, nil
}

func latestRelease(ctx context.Context, client *http.Client, repo, baseURL string) (release githubRelease, err error) {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	url := fmt.Sprintf("%s/repos/%s/releases/latest", strings.TrimRight(baseURL, "/"), repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return githubRelease{}, fmt.Errorf("build release request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "atb-self-update")

	resp, err := client.Do(req)
	if err != nil {
		return githubRelease{}, fmt.Errorf("fetch latest release: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeWithContext(resp.Body, "close latest release response body"))
	}()

	if resp.StatusCode != http.StatusOK {
		return githubRelease{}, fmt.Errorf("%w: latest release returned %s", errUnexpectedResponse, resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return githubRelease{}, fmt.Errorf("decode latest release: %w", err)
	}

	if release.TagName == "" {
		return githubRelease{}, fmt.Errorf("%w: missing tag name", errUnexpectedResponse)
	}

	return release, nil
}

func fetchVerifyAndReplace(ctx context.Context, client *http.Client, release githubRelease, goos, goarch, executablePath string) error {
	assetName := archiveName(goos, goarch)

	assetURL, err := findAssetURL(release, assetName)
	if err != nil {
		return err
	}

	checksumURL, err := findAssetURL(release, checksumAssetName)
	if err != nil {
		return err
	}

	expectedHash, err := fetchExpectedChecksum(ctx, client, checksumURL, assetName)
	if err != nil {
		return err
	}

	return downloadAndReplace(ctx, client, assetURL, expectedHash, executablePath)
}

func archiveName(goos, goarch string) string {
	return fmt.Sprintf("%s_%s_%s.tar.gz", binaryName, goos, goarch)
}

func findAssetURL(release githubRelease, assetName string) (string, error) {
	for _, asset := range release.Assets {
		if asset.Name == assetName && asset.BrowserDownloadURL != "" {
			return asset.BrowserDownloadURL, nil
		}
	}

	return "", fmt.Errorf("%w: %s", errAssetNotFound, assetName)
}

func fetchExpectedChecksum(ctx context.Context, client *http.Client, checksumURL, assetName string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return "", fmt.Errorf("build checksum request: %w", err)
	}

	req.Header.Set("User-Agent", "atb-self-update")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download checksums: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: checksums returned %s", errUnexpectedResponse, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read checksums: %w", err)
	}

	return parseChecksum(string(body), assetName)
}

func parseChecksum(checksums, assetName string) (string, error) {
	for _, line := range strings.Split(checksums, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == assetName {
			return fields[0], nil
		}
	}

	return "", fmt.Errorf("%w: %s", errChecksumNotFound, assetName)
}

func downloadAndReplace(ctx context.Context, client *http.Client, assetURL, expectedHash, executablePath string) (err error) {
	archivePath, actualHash, err := downloadArchive(ctx, client, assetURL)
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(archivePath) }()

	if actualHash != expectedHash {
		return fmt.Errorf("%w: expected %s, got %s", errChecksumMismatch, expectedHash, actualHash)
	}

	//nolint:gosec // archivePath is a temp file created by downloadArchive, not user input.
	archiveFile, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open verified archive: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeWithContext(archiveFile, "close verified archive"))
	}()

	return replaceExecutable(archiveFile, executablePath)
}

func downloadArchive(ctx context.Context, client *http.Client, assetURL string) (archivePath string, checksum string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL, nil)
	if err != nil {
		return "", "", fmt.Errorf("build asset request: %w", err)
	}

	req.Header.Set("User-Agent", "atb-self-update")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("download release asset: %w", err)
	}
	defer func() {
		err = errors.Join(err, closeWithContext(resp.Body, "close release asset response body"))
	}()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("%w: release asset returned %s", errUnexpectedResponse, resp.Status)
	}

	return saveAndHash(resp.Body)
}

func saveAndHash(body io.Reader) (path string, checksum string, err error) {
	tempFile, err := os.CreateTemp("", ".atb-archive-*")
	if err != nil {
		return "", "", fmt.Errorf("create temp archive: %w", err)
	}

	tempPath := tempFile.Name()
	success := false
	defer func() {
		_ = tempFile.Close()
		if !success {
			_ = os.Remove(tempPath)
		}
	}()

	hasher := sha256.New()
	if _, err := io.Copy(tempFile, io.TeeReader(body, hasher)); err != nil {
		return "", "", fmt.Errorf("write temp archive: %w", err)
	}

	success = true

	return tempPath, hex.EncodeToString(hasher.Sum(nil)), nil
}

func replaceExecutable(archive io.Reader, executablePath string) error {
	mode, err := executableMode(executablePath)
	if err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(filepath.Dir(executablePath), ".atb-update-*")
	if err != nil {
		return fmt.Errorf("create temp executable: %w", err)
	}

	tempPath := tempFile.Name()
	cleanup := true
	tempClosed := false
	defer func() {
		if !tempClosed {
			_ = tempFile.Close()
		}
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if err := extractBinaryFromArchive(archive, tempFile); err != nil {
		return err
	}

	if err := tempFile.Chmod(mode); err != nil {
		return fmt.Errorf("chmod temp executable: %w", err)
	}

	if err := closeWithContext(tempFile, "close temp executable"); err != nil {
		return err
	}
	tempClosed = true

	if err := os.Rename(tempPath, executablePath); err != nil {
		return fmt.Errorf("replace executable %s: %w", executablePath, err)
	}

	cleanup = false

	return nil
}

func executableMode(executablePath string) (os.FileMode, error) {
	stat, err := os.Stat(executablePath)
	if err != nil {
		return 0, fmt.Errorf("stat executable %s: %w", executablePath, err)
	}

	mode := stat.Mode().Perm()
	if mode == 0 {
		mode = 0o755
	}

	return mode, nil
}

func extractBinaryFromArchive(archive io.Reader, tempFile *os.File) (err error) {
	gzipReader, err := gzip.NewReader(archive)
	if err != nil {
		return fmt.Errorf("read gzip archive: %w", err)
	}

	defer func() {
		err = errors.Join(err, closeWithContext(gzipReader, "close gzip reader"))
	}()

	tarReader := tar.NewReader(gzipReader)
	found, err := copyBinaryFromTar(tarReader, tempFile)
	if err != nil {
		return err
	}

	if !found {
		return fmt.Errorf("extract binary from release archive: %w", errAssetNotFound)
	}

	return nil
}

func copyBinaryFromTar(tarReader *tar.Reader, tempFile *os.File) (bool, error) {
	for {
		header, err := tarReader.Next()
		if errors.Is(err, io.EOF) {
			return false, nil
		}

		if err != nil {
			return false, fmt.Errorf("read tar archive: %w", err)
		}

		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != binaryName {
			continue
		}

		if header.Size <= 0 {
			return false, fmt.Errorf("%w: %d", errInvalidBinarySize, header.Size)
		}

		if header.Size > maxReleaseBinarySize {
			return false, fmt.Errorf("%w: %d", errBinaryTooLarge, header.Size)
		}

		if _, err := io.CopyN(tempFile, tarReader, header.Size); err != nil {
			return false, fmt.Errorf("write temp executable: %w", err)
		}

		return true, nil
	}
}

func closeWithContext(closer io.Closer, action string) error {
	if err := closer.Close(); err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}

	return nil
}
