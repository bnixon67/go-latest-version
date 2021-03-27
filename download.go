/*
Copyright 2021 Bill Nixon

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// TeeWriter is a Writer interface used in TeeReader.
type TeeWriter struct {
	Expected          int64     // bytes expected
	ExpectedStrLength int       // length of Expected as a string, used for formatting
	Written           int64     // bytes written
	Hash              hash.Hash // hash of bytes written
}

// NewTeeWriter creates a new TeeWriter.
func NewTeeWriter(expected int64, hash hash.Hash) *TeeWriter {
	return &TeeWriter{
		Expected:          expected,
		ExpectedStrLength: len(strconv.FormatInt(expected, 10)),
		Written:           0,
		Hash:              hash,
	}
}

// Write keeps track of the number of bytes and computes a hash on the fly.
func (tw *TeeWriter) Write(data []byte) (int, error) {
	// write data to hash
	tw.Hash.Write(data)

	// count bytes
	n := len(data)
	tw.Written += int64(n)

	// show progress
	fmt.Printf("\r%s", strings.Repeat(" ", 40))
	fmt.Printf(
		"\r%3.0f%% (%*d of %d) complete",
		100.0*float64(tw.Written)/float64(tw.Expected),
		tw.ExpectedStrLength, tw.Written,
		tw.Expected)

	return n, nil
}

// DownloadFile will download the given file at url and write the file to filepath.
// After download, the expectedSize and hash will be checked.
// If filepath exists, it will be overwritten without warning.
func DownloadFile(url string, filepath string, expectedSize int64, hash hash.Hash) (size int64, hashStr string, err error) {
	// create the file, overwriting any existing file of the same name
	out, err := os.Create(filepath)
	if err != nil {
		return
	}
	defer out.Close()

	// get the content at the given URL
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// use a TeeReader to download the file and also show progress and compute hash on the fly
	teeWriter := NewTeeWriter(expectedSize, hash)
	_, err = io.Copy(out, io.TeeReader(resp.Body, teeWriter))
	if err != nil {
		return
	}

	fmt.Println()

	// set return values
	size = teeWriter.Written
	hashStr = fmt.Sprintf("%x", teeWriter.Hash.Sum(nil))

	return
}
