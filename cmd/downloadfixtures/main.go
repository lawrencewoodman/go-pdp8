/*
 * A utility to download the files needed for testing
 *
 * Copyright (C) 2023 Lawrence Woodman <lwoodman@vlifesystems.com>
 *
 * Licensed under an MIT licence.  Please see LICENCE.md for details.
 */

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

/*
 * The following is a list of files needed for testing along
 * with their possible sources and file sizes
 */
var files = []fileDesc{
	fileDesc{filename: "dec-08-lbaa-pm.rim", sources: []source{source{url: "https://ak6dn.github.io/PDP-8/MAINDEC/Binary_Loaders/decbin.rim", size: 408}, source{url: "http://bitsavers.informatik.uni-stuttgart.de/bits/DEC/pdp8/papertapeImages/set2/tray2/dec-08-lbaa-pm_5-10-67.bin", size: 673}}},
	fileDesc{filename: "maindec-08-d01a-pb.bin", sources: []source{source{url: "http://dustyoldcomputers.com/pdp-common/reference/papertapes/maindec/maindec-08-d01a-pb.bin", size: 4328}}},
	fileDesc{filename: "maindec-08-d02b-pb.bin", sources: []source{source{url: "http://dustyoldcomputers.com/pdp-common/reference/papertapes/maindec/maindec-08-d02b-pb.bin", size: 1876}}},
}

func downloadFile(destinationDir string, desc fileDesc) error {
	var f *os.File
	var err error

	fullDestinationFilename := filepath.Join(destinationDir, desc.filename)
	if _, err = os.Stat(fullDestinationFilename); !os.IsNotExist(err) {
		fmt.Printf("File already exists: %s\n", fullDestinationFilename)
		return nil
	}

	client := http.Client{
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			r.URL.Opaque = r.URL.Path
			return nil
		},
	}

	ok := false
	for _, s := range desc.sources {
		// Get file
		resp, err := client.Get(s.url)
		if err != nil {
			fmt.Printf("Download failed, url: %s, error: %s\n", s.url, err)
			continue
		}
		defer resp.Body.Close()

		// Put contents in a file on filesystem
		f, err = os.Create(fullDestinationFilename)
		if err != nil {
			return err
		}
		defer f.Close()

		size, err := io.Copy(f, resp.Body)
		if err != nil {
			return err
		}

		if size != int64(s.size) {
			if err := os.Remove(fullDestinationFilename); err != nil {
				return err
			}
			fmt.Printf("Download failed, url: %s, error: unexpected file size, got: %d, want: %d\n", s.url, size, s.size)
			continue
		}

		fmt.Printf("Downloaded: %s\n", desc.filename)
		ok = true
		break
	}

	if !ok {
		return fmt.Errorf("all sources failed")
	}

	return nil
}

func usage(errMsg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", errMsg)
	fmt.Fprintf(os.Stderr, "Usage: %s -confirm destinationDir\n", os.Args[0])
}

type fileDesc struct {
	filename string   // The name of the file to create
	sources  []source // Possible sources for the file
}

type source struct {
	// TODO: Should we include a checksum?
	url  string
	size int
}

func main() {

	if len(os.Args) < 2 || os.Args[1] != "-confirm" {
		usage("these files maybe copyrighted, supply -confirm switch to confirm that you are aware")
		os.Exit(1)
	}

	if len(os.Args) < 3 {
		usage("no destination directory")
		os.Exit(1)
	}

	destinationDir := os.Args[2]
	fmt.Printf("Downloading files to: %s\n", destinationDir)

	for _, f := range files {
		err := downloadFile(destinationDir, f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to download: %s, error: %s\n", f.filename, err)
			os.Exit(1)
		}
	}
}
