// Copyright 2023 Bill Nixon. All rights reserved.
// Use of this source code is governed by the license found in the LICENSE file.

package main

import (
	"errors"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"strconv"
)

// ProgressHashWriter combines hash computation with progress display for written bytes.
type ProgressHashWriter struct {
	Expected    int64     // Total expected bytes.
	expectedLen int       // Length of the Expected as a string. Precalculate to avoid repeatedly computing in Write().
	Written     int64     // Total bytes written.
	Hash        hash.Hash // Hash of written bytes.
}

// NewProgressHashWriter initializes a new ProgressHashWriter.
func NewProgressHashWriter(expected int64, h hash.Hash) *ProgressHashWriter {
	return &ProgressHashWriter{
		Expected:    expected,
		expectedLen: len(strconv.FormatInt(expected, 10)),
		Written:     0,
		Hash:        h,
	}
}

// Write tracks and displays progress while updating the hash.
// Use for real-time progress updates and integrity verification during file downloads.
func (tw *ProgressHashWriter) Write(data []byte) (int, error) {
	// Update the hash with new data.
	tw.Hash.Write(data)

	// Update the total bytes written.
	n := len(data)
	tw.Written += int64(n)

	// Display current progress.
	fmt.Printf("\r%3.0f%% (%*d of %d) complete",
		100.0*float64(tw.Written)/float64(tw.Expected),
		tw.expectedLen, tw.Written,
		tw.Expected)

	return n, nil
}

var ErrDownloadFailed = errors.New("download failed")

// DownloadFileWithProgressAndChecksum downloads a file with a progress display and checksum computation.
// It downloads a file from url, saves it to a specified filepath, and returns size and checksum for verification.
// If the file already exists at the filepath, it will be overwritten.
func DownloadFileWithProgressAndChecksum(url, filepath string, expectedSize int64, h hash.Hash) (size int64, checksum string, err error) {
	fmt.Printf("Downloading %q to %q\n", url, filepath)

	// Create or overwrite the file
	out, err := os.Create(filepath)
	if err != nil {
		return 0, "", fmt.Errorf("%w: %w", ErrDownloadFailed, err)
	}
	defer out.Close()

	// Get the content from url.
	resp, err := http.Get(url)
	if err != nil {
		return 0, "", fmt.Errorf("%w: %w", ErrDownloadFailed, err)
	}
	defer resp.Body.Close()

	// Check for successful response.
	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("%w: %q %s", ErrDownloadFailed,
			url, http.StatusText(resp.StatusCode))
	}

	// Initialize the ProgressHashWriter
	teeWriter := NewProgressHashWriter(expectedSize, h)

	// Download the file, displaying progress and computing hash
	_, err = io.Copy(out, io.TeeReader(resp.Body, teeWriter))
	if err != nil {
		return 0, "", fmt.Errorf("%w: %w", ErrDownloadFailed, err)
	}

	fmt.Println()

	// Return the size and checksum of the downloaded file
	size = teeWriter.Written
	checksum = fmt.Sprintf("%x", teeWriter.Hash.Sum(nil))

	return size, checksum, nil
}
