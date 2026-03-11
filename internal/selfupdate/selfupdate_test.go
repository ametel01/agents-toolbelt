package selfupdate

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUpdateReplacesExecutableWhenNewerReleaseExists(t *testing.T) {
	t.Parallel()

	archive := testArchive(t, "new-binary")
	var baseURL string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/atb/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","assets":[{"name":"atb_linux_amd64.tar.gz","browser_download_url":"` + baseURL + `/downloads/atb_linux_amd64.tar.gz"}]}`))
		case "/downloads/atb_linux_amd64.tar.gz":
			_, _ = w.Write(archive)
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
		_, _ = w.Write([]byte(`{"tag_name":"v0.1.0","assets":[{"name":"atb_linux_amd64.tar.gz","browser_download_url":"` + baseURL + `/downloads/atb_linux_amd64.tar.gz"}]}`))
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
