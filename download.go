// Copyright 2023 Bill Nixon
// Licensed under the Apache License, Version 2.0 (the "License").
// See the LICENSE file for the specific language governing permissions
// and limitations under the License.
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

// ProgressHashWriter displays process and computes a hash as bytes are written.
type ProgressHashWriter struct {
	Expected       int64     // bytes expected
	ExpectedStrLen int       // string length of Expected for formatting
	Written        int64     // bytes written
	Hash           hash.Hash // hash of bytes written
}

// NewProgressHashWriter creates a new ProgressHashWriter that
// expects the specified number of bytes to be written and computes a
// checksum using the provided hash algorithm.
func NewProgressHashWriter(expected int64, hash hash.Hash) *ProgressHashWriter {
	return &ProgressHashWriter{
		Expected:       expected,
		ExpectedStrLen: len(strconv.FormatInt(expected, 10)),
		Written:        0,
		Hash:           hash,
	}
}

// Write displays the progress while counting the bytes written and
// computing the hash. This function is designed to be used while
// downloading a file, providing real-time progress updates and allowing
// verification of the downloaded file's integrity.
func (tw *ProgressHashWriter) Write(data []byte) (int, error) {
	// add data to running hash
	tw.Hash.Write(data)

	// update bytes written
	n := len(data)
	tw.Written += int64(n)

	// show progress
	fmt.Printf("\r%3.0f%% (%*d of %d) complete",
		100.0*float64(tw.Written)/float64(tw.Expected),
		tw.ExpectedStrLen, tw.Written,
		tw.Expected)

	return n, nil
}

var ErrDownloadFailed = errors.New("download failed")

// DownloadFileWithProgressAndChecksum downloads a file from the
// specified URL, saves it to the given filepath, and returns the size
// and checksum of the downloaded file for verification.
// The expectedSize is used to display the download progress.
// The checksum is computed using the provided hash.Hash.
// If the filepath already exists, it will be overwritten without warning.
func DownloadFileWithProgressAndChecksum(url, filepath string, expectedSize int64, hash hash.Hash) (size int64, checksum string, err error) {
	fmt.Printf("Downloading %q to %q\n", url, filepath)

	// create the file, overwriting any existing file of the same name
	out, err := os.Create(filepath)
	if err != nil {
		return 0, "", fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	defer out.Close()

	// get the content at the given URL
	resp, err := http.Get(url)
	if err != nil {
		return 0, "", fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("%w: %q %s", ErrDownloadFailed, url, http.StatusText(resp.StatusCode))
	}

	// Use custom Writer to download file, show progress, and compute hash
	teeWriter := NewProgressHashWriter(expectedSize, hash)
	_, err = io.Copy(out, io.TeeReader(resp.Body, teeWriter))
	if err != nil {
		return 0, "", fmt.Errorf("%w: %v", ErrDownloadFailed, err)
	}

	fmt.Println()

	size = teeWriter.Written
	checksum = fmt.Sprintf("%x", teeWriter.Hash.Sum(nil))

	return size, checksum, err
}
