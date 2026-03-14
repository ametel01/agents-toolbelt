package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateReplacesExecutableWhenNewerReleaseExists(t *testing.T) {
	t.Parallel()

	archive := testArchive(t, "new-binary")
	checksums := testChecksums(t, archive, "atb_linux_amd64.tar.gz")
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/atb/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","assets":[` +
				`{"name":"atb_linux_amd64.tar.gz","browser_download_url":"` + baseURL + `/downloads/atb_linux_amd64.tar.gz"},` +
				`{"name":"checksums.txt","browser_download_url":"` + baseURL + `/downloads/checksums.txt"}` +
				`]}`))
		case "/downloads/atb_linux_amd64.tar.gz":
			_, _ = w.Write(archive)
		case "/downloads/checksums.txt":
			_, _ = fmt.Fprint(w, checksums)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	executablePath := filepath.Join(t.TempDir(), "atb")
	writeExecutableFixture(t, executablePath, "old-binary")

	result, err := Update(context.Background(), Options{
		BaseURL:        server.URL,
		Client:         server.Client(),
		CurrentVersion: "v0.1.0",
		ExecutablePath: executablePath,
		GOARCH:         "amd64",
		GOOS:           "linux",
		Repo:           "test/atb",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if !result.Updated {
		t.Fatal("Update() did not mark the binary as updated")
	}

	content := readTempFile(t, executablePath)

	if string(content) != "new-binary" {
		t.Fatalf("updated executable = %q, want %q", string(content), "new-binary")
	}
}

func TestUpdateNoopsWhenAlreadyCurrent(t *testing.T) {
	t.Parallel()

	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/test/atb/releases/latest" {
			http.NotFound(w, r)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v0.1.0","assets":[` +
			`{"name":"atb_linux_amd64.tar.gz","browser_download_url":"` + baseURL + `/downloads/atb_linux_amd64.tar.gz"},` +
			`{"name":"checksums.txt","browser_download_url":"` + baseURL + `/downloads/checksums.txt"}` +
			`]}`))
	}))
	defer server.Close()
	baseURL = server.URL

	executablePath := filepath.Join(t.TempDir(), "atb")
	writeExecutableFixture(t, executablePath, "current-binary")

	result, err := Update(context.Background(), Options{
		BaseURL:        server.URL,
		Client:         server.Client(),
		CurrentVersion: "v0.1.0",
		ExecutablePath: executablePath,
		GOARCH:         "amd64",
		GOOS:           "linux",
		Repo:           "test/atb",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if result.Updated {
		t.Fatal("Update() marked an up-to-date binary as updated")
	}

	content := readTempFile(t, executablePath)

	if string(content) != "current-binary" {
		t.Fatalf("executable changed unexpectedly: %q", string(content))
	}
}

func TestUpdateAcceptsCurrentVersionWithoutVPrefix(t *testing.T) {
	t.Parallel()

	archive := testArchive(t, "new-binary")
	checksums := testChecksums(t, archive, "atb_linux_amd64.tar.gz")
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/atb/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","assets":[` +
				`{"name":"atb_linux_amd64.tar.gz","browser_download_url":"` + baseURL + `/downloads/atb_linux_amd64.tar.gz"},` +
				`{"name":"checksums.txt","browser_download_url":"` + baseURL + `/downloads/checksums.txt"}` +
				`]}`))
		case "/downloads/atb_linux_amd64.tar.gz":
			_, _ = w.Write(archive)
		case "/downloads/checksums.txt":
			_, _ = fmt.Fprint(w, checksums)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	executablePath := filepath.Join(t.TempDir(), "atb")
	writeExecutableFixture(t, executablePath, "old-binary")

	result, err := Update(context.Background(), Options{
		BaseURL:        server.URL,
		Client:         server.Client(),
		CurrentVersion: "0.1.0",
		ExecutablePath: executablePath,
		GOARCH:         "amd64",
		GOOS:           "linux",
		Repo:           "test/atb",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if !result.Updated {
		t.Fatal("Update() did not mark the binary as updated")
	}

	if result.CurrentVersion != "v0.1.0" {
		t.Fatalf("result.CurrentVersion = %q, want %q", result.CurrentVersion, "v0.1.0")
	}
}

func TestUpdateAcceptsLatestVersionWithoutVPrefix(t *testing.T) {
	t.Parallel()

	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/test/atb/releases/latest" {
			http.NotFound(w, r)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"0.1.0","assets":[` +
			`{"name":"atb_linux_amd64.tar.gz","browser_download_url":"` + baseURL + `/downloads/atb_linux_amd64.tar.gz"},` +
			`{"name":"checksums.txt","browser_download_url":"` + baseURL + `/downloads/checksums.txt"}` +
			`]}`))
	}))
	defer server.Close()
	baseURL = server.URL

	executablePath := filepath.Join(t.TempDir(), "atb")
	writeExecutableFixture(t, executablePath, "current-binary")

	result, err := Update(context.Background(), Options{
		BaseURL:        server.URL,
		Client:         server.Client(),
		CurrentVersion: "v0.1.0",
		ExecutablePath: executablePath,
		GOARCH:         "amd64",
		GOOS:           "linux",
		Repo:           "test/atb",
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	if result.Updated {
		t.Fatal("Update() marked an up-to-date binary as updated")
	}

	if result.LatestVersion != "v0.1.0" {
		t.Fatalf("result.LatestVersion = %q, want %q", result.LatestVersion, "v0.1.0")
	}
}

func TestUpdateRejectsNonReleaseVersions(t *testing.T) {
	t.Parallel()

	executablePath := filepath.Join(t.TempDir(), "atb")
	writeExecutableFixture(t, executablePath, "dev-binary")

	_, err := Update(context.Background(), Options{
		CurrentVersion: "dev",
		ExecutablePath: executablePath,
		GOARCH:         "amd64",
		GOOS:           "linux",
	})
	if err == nil {
		t.Fatal("Update() error = nil, want invalid version failure")
	}
}

func TestUpdateFailsOnChecksumMismatch(t *testing.T) {
	t.Parallel()

	archive := testArchive(t, "new-binary")
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/atb/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","assets":[` +
				`{"name":"atb_linux_amd64.tar.gz","browser_download_url":"` + baseURL + `/downloads/atb_linux_amd64.tar.gz"},` +
				`{"name":"checksums.txt","browser_download_url":"` + baseURL + `/downloads/checksums.txt"}` +
				`]}`))
		case "/downloads/atb_linux_amd64.tar.gz":
			_, _ = w.Write(archive)
		case "/downloads/checksums.txt":
			_, _ = fmt.Fprint(w, "badhash0000000000000000000000000000000000000000000000000000000000  atb_linux_amd64.tar.gz\n")
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	baseURL = server.URL

	executablePath := filepath.Join(t.TempDir(), "atb")
	writeExecutableFixture(t, executablePath, "old-binary")

	_, err := Update(context.Background(), Options{
		BaseURL:        server.URL,
		Client:         server.Client(),
		CurrentVersion: "v0.1.0",
		ExecutablePath: executablePath,
		GOARCH:         "amd64",
		GOOS:           "linux",
		Repo:           "test/atb",
	})
	if !errors.Is(err, errChecksumMismatch) {
		t.Fatalf("Update() error = %v, want %v", err, errChecksumMismatch)
	}

	content := readTempFile(t, executablePath)
	if string(content) != "old-binary" {
		t.Fatalf("executable changed despite checksum mismatch: %q", string(content))
	}
}

func TestUpdateFailsWhenChecksumAssetMissing(t *testing.T) {
	t.Parallel()

	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/test/atb/releases/latest" {
			http.NotFound(w, r)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","assets":[` +
			`{"name":"atb_linux_amd64.tar.gz","browser_download_url":"` + baseURL + `/downloads/atb_linux_amd64.tar.gz"}` +
			`]}`))
	}))
	defer server.Close()
	baseURL = server.URL

	executablePath := filepath.Join(t.TempDir(), "atb")
	writeExecutableFixture(t, executablePath, "old-binary")

	_, err := Update(context.Background(), Options{
		BaseURL:        server.URL,
		Client:         server.Client(),
		CurrentVersion: "v0.1.0",
		ExecutablePath: executablePath,
		GOARCH:         "amd64",
		GOOS:           "linux",
		Repo:           "test/atb",
	})
	if !errors.Is(err, errAssetNotFound) {
		t.Fatalf("Update() error = %v, want %v", err, errAssetNotFound)
	}
}

func TestParseChecksum(t *testing.T) {
	t.Parallel()

	checksums := "abc123  atb_linux_amd64.tar.gz\ndef456  atb_darwin_arm64.tar.gz\n"

	hash, err := parseChecksum(checksums, "atb_linux_amd64.tar.gz")
	if err != nil {
		t.Fatalf("parseChecksum() error = %v", err)
	}

	if hash != "abc123" {
		t.Fatalf("parseChecksum() = %q, want %q", hash, "abc123")
	}
}

func TestParseChecksumReturnsErrorForMissingAsset(t *testing.T) {
	t.Parallel()

	checksums := "abc123  atb_linux_amd64.tar.gz\n"

	_, err := parseChecksum(checksums, "atb_darwin_arm64.tar.gz")
	if !errors.Is(err, errChecksumNotFound) {
		t.Fatalf("parseChecksum() error = %v, want %v", err, errChecksumNotFound)
	}
}

func testArchive(t *testing.T, binaryContent string) []byte {
	t.Helper()

	tempFile, err := os.CreateTemp(t.TempDir(), "archive-*.tar.gz")
	if err != nil {
		t.Fatalf("os.CreateTemp() error = %v", err)
	}
	defer func() {
		if err := os.Remove(tempFile.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Errorf("os.Remove() error = %v", err)
		}
	}()

	gzipWriter := gzip.NewWriter(tempFile)
	tarWriter := tar.NewWriter(gzipWriter)
	if err := tarWriter.WriteHeader(&tar.Header{
		Name: "atb",
		Mode: 0o755,
		Size: int64(len(binaryContent)),
	}); err != nil {
		t.Fatalf("tarWriter.WriteHeader() error = %v", err)
	}

	if _, err := tarWriter.Write([]byte(binaryContent)); err != nil {
		t.Fatalf("tarWriter.Write() error = %v", err)
	}

	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tarWriter.Close() error = %v", err)
	}

	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzipWriter.Close() error = %v", err)
	}

	if err := tempFile.Close(); err != nil {
		t.Fatalf("tempFile.Close() error = %v", err)
	}

	return readTempFile(t, tempFile.Name())
}

func testChecksums(t *testing.T, archive []byte, assetName string) string {
	t.Helper()

	hasher := sha256.New()
	hasher.Write(archive)

	return hex.EncodeToString(hasher.Sum(nil)) + "  " + assetName + "\n"
}

func writeExecutableFixture(t *testing.T, executablePath, content string) {
	t.Helper()

	if err := os.WriteFile(executablePath, []byte(content), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v", err)
	}
}

func readTempFile(t *testing.T, path string) []byte {
	t.Helper()

	//nolint:gosec // The path comes from the test temp directory.
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile() error = %v", err)
	}

	return content
}
