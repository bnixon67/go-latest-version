package main

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// TeeWriter is a Writer interface used in TeeReader
type TeeWriter struct {
	Expected          int64     // bytes expected
	ExpectedStrLength int       // length of Expected as a string, used for formatting
	Written           int64     // bytes written
	Hash              hash.Hash // hash of bytes written
}

func NewTeeWriter(expected int64) *TeeWriter {
	return &TeeWriter{
		Expected:          expected,
		ExpectedStrLength: len(strconv.FormatInt(expected, 10)),
		Written:           0,
		Hash:              sha256.New(),
	}
}

// Write keeps track of the number of bytes and computes a hash on the fly
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

func DownloadFile(url string, filepath string, expectedSize int64) (size int64, hashStr string, err error) {

	out, err := os.Create(filepath)
	if err != nil {
		return
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	counter := NewTeeWriter(expectedSize)
	_, err = io.Copy(out, io.TeeReader(resp.Body, counter))
	if err != nil {
		return
	}

	fmt.Println()

	size = counter.Written
	hashStr = fmt.Sprintf("%x", counter.Hash.Sum(nil))

	return
}
