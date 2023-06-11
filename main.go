// Copyright 2023 Bill Nixon
// Licensed under the Apache License, Version 2.0 (the "License").
// See the LICENSE file for the specific language governing permissions
// and limitations under the License.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"runtime"
)

// ReleaseFile represents a file on the go.dev downloads page.
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

// ReleaseInfo represents release downloads from golang.org/dl/?mode=json.
// See https://pkg.go.dev/golang.org/x/website/internal/dl#Release
type ReleaseInfo []struct {
	Version string        `json:"version"`
	Stable  bool          `json:"stable"`
	Files   []ReleaseFile `json:"files"`
}

const (
	releaseURL        = "https://golang.org/dl/?mode=json"
	downloadPrefixURL = "https://golang.org/dl/"
)

// getReleaseInfo retrieves the release information from the url.
func getReleaseInfo(url string) (ReleaseInfo, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil,
			fmt.Errorf("getReleaseInfo http.Get failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil,
			fmt.Errorf("getReleaseInfo http.Get failed: %q %s",
				url, http.StatusText(resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil,
			fmt.Errorf("getReleaseInfo io.ReadAll failed: %w", err)
	}

	var releaseInfo ReleaseInfo

	err = json.Unmarshal(body, &releaseInfo)
	if err != nil {
		return nil,
			fmt.Errorf("getReleaseInfo UnMarshal failed: %w", err)
	}

	return releaseInfo, nil
}

// findMatchingReleaseFile searches for a release file in the release
// info that matches the current OS and architecture.
func findMatchingReleaseFile(releaseInfo ReleaseInfo) (ReleaseFile, error) {
	kind := "archive"

	// for windows and darwin, prefer installer over archive
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		kind = "installer"
	}

	for _, r := range releaseInfo {
		for _, file := range r.Files {
			if file.OS == runtime.GOOS && file.Arch == runtime.GOARCH && file.Kind == kind {
				return file, nil
			}
		}
	}

	return ReleaseFile{}, fmt.Errorf("no matching file found")
}

// downloadAndVerifyFile downloads and verifies the release file.
func downloadAndVerifyFile(file ReleaseFile) error {
	fullURL, err := url.JoinPath(downloadPrefixURL, file.Filename)
	if err != nil {
		return err
	}

	size, checksum, err := DownloadFileWithProgressAndChecksum(fullURL, file.Filename, file.Size, sha256.New())
	if err != nil {
		return err
	}

	if file.SHA256 != checksum {
		return fmt.Errorf("SHA256 checksum mismatch: got %v want %v",
			checksum, file.SHA256)
	}

	if file.Size != size {
		return fmt.Errorf("file size mismatch: got %v want %v",
			size, file.Size)
	}

	return nil
}

func main() {
	fmt.Printf("Running: %s on %s.%s\n",
		runtime.Version(), runtime.GOOS, runtime.GOARCH)

	releaseInfo, err := getReleaseInfo(releaseURL)
	if err != nil {
		fmt.Println(err)

		return
	}

	file, err := findMatchingReleaseFile(releaseInfo)
	if err != nil {
		fmt.Println(err)

		return
	}

	fmt.Printf("Latest : %s on %s.%s\n",
		file.Version, file.OS, file.Arch)

	if file.Version == runtime.Version() {
		fmt.Println("Running current version.")

		return
	}

	err = downloadAndVerifyFile(file)
	if err != nil {
		fmt.Printf("Download failed: %v\n", err)

		return
	}

	if runtime.GOOS != "windows" && runtime.GOOS != "darwin" {
		fmt.Println("Run the following command to install:")
		fmt.Printf("sudo -- sh -c \"rm -rf /usr/local/go && tar -C /usr/local -xzf %s\"\n", file.Filename)
	}
}
