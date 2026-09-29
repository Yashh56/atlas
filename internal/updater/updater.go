package updater

import (
	"archive/tar"
	"archive/zip"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
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

const repo = "Yashh56/atlas"
const apiURL = "https://api.github.com/repos/" + repo + "/releases/latest"

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckUpdate queries GitHub for the latest release and compares it with the current version.
// It returns the latest Release if an update is available.
func CheckUpdate(currentVersion string) (*Release, error) {
	if currentVersion == "dev" {
		return nil, nil // No updates checked for dev builds
	}

	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}

	latestVersion := strings.TrimPrefix(rel.TagName, "v")
	currVer := strings.TrimPrefix(currentVersion, "v")

	if latestVersion != currVer {
		return &rel, nil
	}
	return nil, nil
}

// PerformUpdate downloads and extracts the release, replacing the current executable.
func PerformUpdate(rel *Release) error {
	osName, archName, ext := getPlatformInfo()

	var assetURL string
	var assetName string
	var checksumURL string

	for _, a := range rel.Assets {
		if strings.Contains(a.Name, osName) && strings.Contains(a.Name, archName) && strings.HasSuffix(a.Name, ext) {
			assetURL = a.BrowserDownloadURL
			assetName = a.Name
		}
		if a.Name == "checksums.txt" {
			checksumURL = a.BrowserDownloadURL
		}
	}

	if assetURL == "" {
		return fmt.Errorf("could not find a suitable release asset for %s %s", osName, archName)
	}

	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return err
	}
	exeDir := filepath.Dir(exePath)

	// Create a temp directory inside the same directory as the executable
	// to ensure they are on the same disk drive (prevents cross-device rename errors).
	tempDir, err := os.MkdirTemp(exeDir, ".atlas-update-*")
	if err != nil {
		// Fallback to system temp directory if we don't have write permissions in the exe directory
		tempDir, err = os.MkdirTemp("", "atlas-update-*")
		if err != nil {
			return err
		}
	}
	defer os.RemoveAll(tempDir)

	archivePath := filepath.Join(tempDir, "archive"+ext)

	// Download the release asset
	if err := downloadFile(assetURL, archivePath); err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}

	// Verify Checksum if checksums.txt is available
	if checksumURL != "" {
		checksumPath := filepath.Join(tempDir, "checksums.txt")
		if err := downloadFile(checksumURL, checksumPath); err == nil {
			if err := verifyChecksum(archivePath, assetName, checksumPath); err != nil {
				return fmt.Errorf("checksum verification failed: %w", err)
			}
		}
	}

	extractedBin := filepath.Join(tempDir, "atlas")
	if runtime.GOOS == "windows" {
		extractedBin += ".exe"
	}

	if ext == ".zip" {
		if err := extractZip(archivePath, extractedBin); err != nil {
			return err
		}
	} else {
		if err := extractTarGz(archivePath, extractedBin); err != nil {
			return err
		}
	}

	// Replace binary safely
	oldExe := exePath + ".old"
	_ = os.Remove(oldExe)

	// We use os.Rename which works reliably only when both paths are on the same disk drive
	if err := os.Rename(exePath, oldExe); err != nil {
		// If rename fails (e.g. cross-device link fallback still triggering), try io.Copy fallback
		return fmt.Errorf("failed to backup current executable: %w", err)
	}

	if err := os.Rename(extractedBin, exePath); err != nil {
		// Try to revert rename
		_ = os.Rename(oldExe, exePath)
		return fmt.Errorf("failed to install new executable: %w", err)
	}

	if runtime.GOOS != "windows" {
		_ = os.Chmod(exePath, 0755)
	}

	return nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, resp.Body)
	return err
}

func verifyChecksum(archivePath, assetName, checksumPath string) error {
	// Calculate SHA256 of downloaded archive
	f, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actualHash := hex.EncodeToString(h.Sum(nil))

	// Read checksums.txt to find expected hash
	cf, err := os.Open(checksumPath)
	if err != nil {
		return err
	}
	defer cf.Close()

	scanner := bufio.NewScanner(cf)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Match the exact filename in the checksums.txt output (usually formatted as "hash  filename")
		if strings.HasSuffix(line, assetName) {
			parts := strings.Fields(line)
			if len(parts) > 0 {
				expectedHash := parts[0]
				if actualHash != expectedHash {
					return fmt.Errorf("expected %s, got %s", expectedHash, actualHash)
				}
				return nil // Checksum matched
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	return fmt.Errorf("checksum for %s not found in checksums.txt", assetName)
}

func getPlatformInfo() (osName, archName, ext string) {
	switch runtime.GOOS {
	case "windows":
		osName = "Windows"
		ext = ".zip"
	case "darwin":
		osName = "Darwin"
		ext = ".tar.gz"
	default:
		osName = "Linux"
		ext = ".tar.gz"
	}

	switch runtime.GOARCH {
	case "arm64":
		archName = "arm64"
	default:
		archName = "x86_64"
	}
	return
}

func extractZip(zipPath, dest string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		if strings.HasSuffix(f.Name, "atlas") || strings.HasSuffix(f.Name, "atlas.exe") {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			out, err := os.Create(dest)
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, rc)
			return err
		}
	}
	return fmt.Errorf("atlas executable not found in zip")
}

func extractTarGz(tarPath, dest string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if strings.HasSuffix(hdr.Name, "atlas") {
			out, err := os.Create(dest)
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, tr)
			return err
		}
	}
	return fmt.Errorf("atlas executable not found in tar.gz")
}
