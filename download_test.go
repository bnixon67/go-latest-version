package main

import (
	"crypto/sha256"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadFileWithProgressAndChecksum(t *testing.T) {
	// mock HTTP response and return named files from testdata directory
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		filePath := filepath.Join("testdata/", r.URL.Path)

		data, err := ioutil.ReadFile(filePath)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write(data)
	}))
	defer server.Close()

	// create a temp file
	tempFile, err := os.CreateTemp("", "testfile")
	if err != nil {
		t.Fatalf("cannot create temp file: %v", err)
	}
	tempFileName := tempFile.Name()
	defer os.Remove(tempFileName)

	testCases := []struct {
		name             string
		url              string
		filepath         string
		expectedSize     int64
		expectedChecksum string
		expectedError    error
	}{
		{
			name:             "Valid 0B file",
			url:              server.URL + "/testfile_0B",
			filepath:         tempFileName,
			expectedSize:     0,
			expectedChecksum: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			expectedError:    nil,
		},
		{
			name:             "Valid 1B file",
			url:              server.URL + "/testfile_1B",
			filepath:         tempFileName,
			expectedSize:     1,
			expectedChecksum: "85f97e04d754c81dac21f0ce857adc81170d08c6cfef7cf90edbbabf39d9671a",
			expectedError:    nil,
		},
		{
			name:             "Valid 1MB file",
			url:              server.URL + "/testfile_1MB",
			filepath:         tempFileName,
			expectedSize:     int64(1024 * 1024),
			expectedChecksum: "a7d95f3a178d5133ca7f918e98e880b00217b51a43c47f558568606d6dd7727e",
			expectedError:    nil,
		},
		{
			name:             "Valid file",
			url:              server.URL + "/testfile_x",
			filepath:         tempFileName,
			expectedSize:     1234567,
			expectedChecksum: "50e28566c25df37c47b8c58a9e2d5bd02b394cb21c640702175edc5fafcb9f0c",
			expectedError:    nil,
		},
		{
			name:          "Invalid url",
			url:           "invalidurl",
			filepath:      tempFileName,
			expectedError: ErrDownloadFailed,
		},
		{
			name:          "No such download",
			url:           server.URL + "/nosuchfile",
			filepath:      tempFileName,
			expectedError: ErrDownloadFailed,
		},
		{
			name:          "Invalid filepath",
			url:           server.URL + "/testfile_0B",
			filepath:      "/invalid/path/to/file.txt",
			expectedError: ErrDownloadFailed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			size, checksum, err := DownloadFileWithProgressAndChecksum(tc.url, tc.filepath, tc.expectedSize, sha256.New())

			if !errors.Is(err, tc.expectedError) {
				t.Errorf("Unexpected error.\n Got: %v\nWant: %v", err, tc.expectedError)
			}

			if checksum != tc.expectedChecksum {
				t.Errorf("Unexpected checksum.\n Got: %q\nWant: %q", checksum, tc.expectedChecksum)
			}

			if size != tc.expectedSize {
				t.Errorf("Unexpected size.\n Got: %d\nWant: %d", size, tc.expectedSize)
			}
		})
	}
}
