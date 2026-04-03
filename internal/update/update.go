package update

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	defaultRepo   = "nsourov/gcai"
	binaryName    = "gcai"
	githubRelease = "https://github.com/%s/releases/download/%s/%s"
	apiLatest     = "https://api.github.com/repos/%s/releases/latest"
)

var httpClient = &http.Client{Timeout: 120 * time.Second}

type releaseJSON struct {
	TagName string `json:"tag_name"`
}

// Run downloads the latest GitHub release asset for this OS/arch and replaces the current executable.
// On success it returns the release tag (e.g. "v1.2.0").
func Run(repo string) (tag string, err error) {
	if repo == "" {
		repo = defaultRepo
	}
	goos, goarch, err := goPlatform()
	if err != nil {
		return "", err
	}

	tag, err = fetchLatestTag(repo)
	if err != nil {
		return "", err
	}
	ver := strings.TrimPrefix(tag, "v")
	asset := fmt.Sprintf("%s_%s_%s_%s.tar.gz", binaryName, ver, goos, goarch)
	url := fmt.Sprintf(githubRelease, repo, tag, asset)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("download %s: HTTP %d %s", url, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	bin, err := extractBinary(resp.Body, binaryName)
	if err != nil {
		return "", err
	}

	dest, err := os.Executable()
	if err != nil {
		return "", err
	}
	dest, err = filepath.EvalSymlinks(dest)
	if err != nil {
		dest, _ = os.Executable()
	}

	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, "."+binaryName+".next-*")
	if err != nil {
		return "", fmt.Errorf("temp file: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }
	defer cleanup()

	if _, err := tmp.Write(bin); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Chmod(0o755); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}

	if err := os.Rename(tmpPath, dest); err != nil {
		return "", fmt.Errorf("replace %s: %w (try running with sudo if installed under /usr/local/bin)", dest, err)
	}
	return tag, nil
}

func goPlatform() (goos, goarch string, err error) {
	goos = runtime.GOOS
	goarch = runtime.GOARCH
	switch goos {
	case "darwin", "linux":
	default:
		return "", "", fmt.Errorf("unsupported OS for self-update: %s (use install script or build from source)", goos)
	}
	switch goarch {
	case "amd64", "arm64":
	default:
		return "", "", fmt.Errorf("unsupported architecture for self-update: %s", goarch)
	}
	return goos, goarch, nil
}

func fetchLatestTag(repo string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf(apiLatest, repo), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("releases/latest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("releases/latest: HTTP %d", resp.StatusCode)
	}
	var rel releaseJSON
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("parse release: %w", err)
	}
	if strings.TrimSpace(rel.TagName) == "" {
		return "", fmt.Errorf("empty tag_name in release response")
	}
	return rel.TagName, nil
}

func extractBinary(r io.Reader, wantName string) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("tar: %w", err)
		}
		if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeRegA {
			continue
		}
		base := filepath.Base(h.Name)
		if base != wantName {
			continue
		}
		b, err := io.ReadAll(io.LimitReader(tr, 64<<20))
		if err != nil {
			return nil, err
		}
		return b, nil
	}
	return nil, fmt.Errorf("archive does not contain %q", wantName)
}
