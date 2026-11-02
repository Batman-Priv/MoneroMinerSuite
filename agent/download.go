package main

import (
	"archive/tar"
	"archive/zip"
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

type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type githubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []githubAsset `json:"assets"`
}

func fetchLatestXMRigRelease() (*githubRelease, error) {
	client := http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", "https://api.github.com/repos/xmrig/xmrig/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "miner-agent")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}
	var rel githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

func pickXMRigAsset(rel *githubRelease) (*githubAsset, error) {
	var candidates []string
	switch runtime.GOOS {
	case "linux":
		if runtime.GOARCH == "arm64" {
			candidates = []string{"linux-static-arm64.tar.gz"}
		} else {
			candidates = []string{"linux-static-x64.tar.gz"}
		}
	case "windows":
		if runtime.GOARCH == "arm64" {
			candidates = []string{"windows-arm64.zip"}
		} else {
			
			candidates = []string{"windows-x64.zip", "windows-gcc-x64.zip", "msvc-win64.zip", "win64.zip", "gcc-win64.zip"}
		}
	case "darwin":
		if runtime.GOARCH == "arm64" {
			candidates = []string{"macos-arm64.tar.gz"}
		} else {
			candidates = []string{"macos-x64.tar.gz"}
		}
	default:
		return nil, fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	for _, suffix := range candidates {
		for i := range rel.Assets {
			
			if strings.Contains(rel.Assets[i].Name, suffix) {
				return &rel.Assets[i], nil
			}
		}
	}
	return nil, fmt.Errorf("no matching asset found for %s/%s in release %s", runtime.GOOS, runtime.GOARCH, rel.TagName)
}

func downloadToFile(url, dest string) error {
	client := http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: status %d", resp.StatusCode)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func extractArchive(archivePath, destDir string) error {
	if strings.HasSuffix(archivePath, ".zip") {
		return extractZip(archivePath, destDir)
	}
	return extractTarGz(archivePath, destDir)
}

func safeJoin(destDir, name string) (string, error) {
	target := filepath.Join(destDir, name)
	cleanDest := filepath.Clean(destDir)
	if target != cleanDest && !strings.HasPrefix(target, cleanDest+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe path inside archive: %s", name)
	}
	return target, nil
}

func extractZip(path, destDir string) error {
	r, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target, err := safeJoin(destDir, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		_, err = io.Copy(out, rc)
		out.Close()
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(path, destDir string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
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
		target, err := safeJoin(destDir, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			out.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func InstallXMRig(installDir string) (string, string, error) {
	rel, err := fetchLatestXMRigRelease()
	if err != nil {
		return "", "", fmt.Errorf("failed to read the latest release from GitHub: %w", err)
	}
	asset, err := pickXMRigAsset(rel)
	if err != nil {
		return "", "", err
	}

	tmpDir, err := os.MkdirTemp("", "xmrig-download-*")
	if err != nil {
		return "", "", err
	}
	defer os.RemoveAll(tmpDir)

	archivePath := filepath.Join(tmpDir, asset.Name)
	if err := downloadToFile(asset.BrowserDownloadURL, archivePath); err != nil {
		return "", "", fmt.Errorf("failed to download %s: %w", asset.Name, err)
	}

	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return "", "", err
	}
	if err := extractArchive(archivePath, extractDir); err != nil {
		return "", "", fmt.Errorf("failed to extract the archive: %w", err)
	}

	binName := "xmrig"
	if runtime.GOOS == "windows" {
		binName = "xmrig.exe"
	}
	var found string
	_ = filepath.Walk(extractDir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && info.Name() == binName {
			found = p
		}
		return nil
	})
	if found == "" {
		return "", "", fmt.Errorf("could not find the %s binary inside the archive", binName)
	}

	dest := filepath.Join(installDir, binName)
	if err := copyExecutable(found, dest); err != nil {
		return "", "", err
	}
	return dest, rel.TagName, nil
}
