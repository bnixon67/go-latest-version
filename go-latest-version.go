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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
)

// golang struct of json from golang.org/dl/?mode=json
type VersionInfo []struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
	Files   []struct {
		Filename string `json:"filename"`
		Os       string `json:"os"`
		Arch     string `json:"arch"`
		Version  string `json:"version"`
		Sha256   string `json:"sha256"`
		Size     int64  `json:"size"`
		Kind     string `json:"kind"`
	} `json:"files"`
}

// URL to get the golang version info in JSON format
const dlURL = "https://golang.org/dl/?mode=json"

// URL prefix to download golang file
const dlURLPrefix = "https://dl.google.com/go/"

func main() {
	// get download info from golang.org in json format
	resp, err := http.Get(dlURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)

	// check to make sure we have a json array
	if body[0] != '[' {
		fmt.Println("Invalid json returned from", dlURL)
		return
	}

	// convert json to struct
	var versionInfo VersionInfo
	err = json.Unmarshal(body, &versionInfo)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("current version %s on %s.%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)

	// for most os, download archive file, but for windows download installer file
	kind := "archive"
	if runtime.GOOS == "windows" {
		kind = "installer"
	}

	// first array element is the latest version
	// loop thru Files to find matching GOOS and GOARCH
	for _, file := range versionInfo[0].Files {
		if (file.Os == runtime.GOOS) && (file.Arch == runtime.GOARCH) && (file.Kind == kind) {
			fmt.Printf("latest  version %s on %s.%s\n", file.Version, file.Os, file.Arch)
			fmt.Println()

			if file.Version == runtime.Version() {
				fmt.Println("Already at current version")
			} else {
				// Download latest file
				size, hashStr, err := DownloadFile(dlURLPrefix+file.Filename,
					file.Filename, file.Size, sha256.New())
				if err != nil {
					fmt.Println(err)
					return
				}

				fmt.Println("Downloaded", file.Filename)
				fmt.Println()

				fmt.Printf("Expected sha256 hash %s\n", file.Sha256)
				fmt.Printf("Received sha256 hash %s\n", hashStr)
				if file.Sha256 == hashStr {
					fmt.Println("OK\n")
				} else {
					fmt.Println("FAILED\n")
				}

				fmt.Printf("Expected file size %d\n", file.Size)
				fmt.Printf("Received file size %d\n", size)
				if file.Size == size {
					fmt.Println("OK")
				} else {
					fmt.Println("FAILED")
				}

				fmt.Println("Run the following command to install:")
				fmt.Printf("sudo -- sh -c \"rm -rf /usr/local/go && tar -C /usr/local -xzf %s\"\n", file.Filename)
			}
		}
	}

}
