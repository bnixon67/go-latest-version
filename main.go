// Copyright 2025 Bill Nixon. All rights reserved.
// Use of this source code is governed by the license found in the LICENSE file.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"time"
)

// ReleaseFile represents a file available on the go.dev downloads page.
// See https://pkg.go.dev/golang.org/x/website/internal/dl#File
type ReleaseFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"`
}

// ReleaseInfo represents a collection of Go releases with associated files.
// See https://pkg.go.dev/golang.org/x/website/internal/dl#Release
type ReleaseInfo []struct {
	Version string        `json:"version"`
	Stable  bool          `json:"stable"`
	Files   []ReleaseFile `json:"files"`
}

const (
	downloadPrefixURL  = "https://go.dev/dl"
	releaseURL         = downloadPrefixURL + "/?mode=json"
	httpClientTimeout  = 30 * time.Second
	downloadMaxRetries = 3
)

// getReleaseInfo gets the latest Go release information from the official URL.
// It returns a ReleaseInfo object containing details about available releases.
func getReleaseInfo(releaseURL string) (ReleaseInfo, error) {
	httpClient := &http.Client{Timeout: httpClientTimeout}
	resp, err := httpClient.Get(releaseURL)
	if err != nil {
		return nil,
			fmt.Errorf("failed to get release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil,
			fmt.Errorf("failed to get release info: %q %s",
				releaseURL, http.StatusText(resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil,
			fmt.Errorf("failed to read release info: %w", err)
	}

	var releaseInfo ReleaseInfo
	err = json.Unmarshal(body, &releaseInfo)
	if err != nil {
		return nil,
			fmt.Errorf("failed to unmarshal release info: %w", err)
	}

	return releaseInfo, nil
}

// findMatchingReleaseFile returns the release file for the current system's OS and architecture.
func findMatchingReleaseFile(releaseInfo ReleaseInfo) (ReleaseFile, error) {
	kind := "archive"
	// for windows and darwin, prefer installer over archive
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		kind = "installer"
	}

	// Search through all releases, not just the first one
	for _, release := range releaseInfo {
		if !release.Stable {
			continue
		}
		for _, file := range release.Files {
			if file.OS == runtime.GOOS && file.Arch == runtime.GOARCH && file.Kind == kind {
				return file, nil
			}
		}
	}

	return ReleaseFile{}, fmt.Errorf("no matching file found for OS: %s, Arch: %s", runtime.GOOS, runtime.GOARCH)
}

// downloadAndVerifyFile downloads a Go release file and verifies its integrity.
// It checks the SHA256 checksum and file size against the provided metadata.
// If the file already exists with correct checksum and size, skips download.
func downloadAndVerifyFile(file ReleaseFile, forceDownload bool) error {
	// Check if file already exists and verify it
	if !forceDownload {
		if fileInfo, err := os.Stat(file.Filename); err == nil {
			if fileInfo.Size() == file.Size {
				// Verify checksum of existing file
				f, err := os.Open(file.Filename)
				if err == nil {
					defer f.Close()
					h := sha256.New()
					if _, err := io.Copy(h, f); err == nil {
						existingChecksum := fmt.Sprintf("%x", h.Sum(nil))
						if existingChecksum == file.SHA256 {
							fmt.Printf("File %q already exists with correct checksum, skipping download.\n", file.Filename)
							return nil
						}
					}
				}
			}
		}
	}

	fullURL, err := url.JoinPath(downloadPrefixURL, file.Filename)
	if err != nil {
		return fmt.Errorf("failed to join path: %w", err)
	}

	size, checksum, err := DownloadFileWithProgressAndChecksum(fullURL, file.Filename, file.Size, sha256.New())
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	if file.SHA256 != checksum {
		return fmt.Errorf("checksum incorrect: got %v want %v",
			checksum, file.SHA256)
	}

	if file.Size != size {
		return fmt.Errorf("file size incorrect: got %v want %v",
			size, file.Size)
	}

	return nil
}

const (
	ExitErrReleaseInfo = 1
	ExitErrMatchFile   = 2
	ExitErrDownload    = 3
)

func main() {
	// Define and parse the forceDownload flag.
	var forceDownload bool
	flag.BoolVar(&forceDownload, "force", false, "Force download of the latest Go release")
	flag.Parse()

	fmt.Printf("Running %s on %s/%s\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)

	releaseInfo, err := getReleaseInfo(releaseURL)
	if err != nil {
		fmt.Printf("Error getting release info: %v\n", err)
		os.Exit(ExitErrReleaseInfo)
	}

	file, err := findMatchingReleaseFile(releaseInfo)
	if err != nil {
		fmt.Printf("Error finding matching release file: %v\n", err)
		os.Exit(ExitErrMatchFile)
	}

	fmt.Printf("Latest  %s on %s/%s\n",
		file.Version, file.OS, file.Arch)

	// Check if the current version running and if forceDownload is not set.
	if file.Version == runtime.Version() && !forceDownload {
		fmt.Println("Running current version. Use -force to override.")
		return
	}

	err = downloadAndVerifyFile(file, forceDownload)
	if err != nil {
		fmt.Printf("Download failed: %v\n", err)
		os.Exit(ExitErrDownload)
	}

	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		fmt.Println("Run the following command to install:")
		fmt.Printf("sudo -- sh -c \"rm -rf /usr/local/go && tar -C /usr/local -xzf %s\"\n", file.Filename)
	}
}
