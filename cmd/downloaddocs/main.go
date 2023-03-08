/*
 * A utility to download the documentation associated with
 * the testing files
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
	fileDesc{filename: "dec-08-lbaa-d.pdf", sources: []source{source{url: "https://deramp.com/downloads/mfe_archive/011-Digital%20Equipment%20Corporation/01%20DEC%20PDP-8%20Family%20Software/02%20Paper%20Tape%20Utilities/DEC-08-LBAA%20Binary%20Loader/DEC-08-LBAA-D%20Binary%20Loader.pdf", size: 526385}}},
	fileDesc{filename: "maindec-08-d01a-d.pdf", sources: []source{source{url: "https://svn.so-much-stuff.com/svn/trunk/pdp8/src/dec/maindec-08-d01/maindec-08-d01a-d.pdf", size: 14138524}}},
	fileDesc{filename: "maindec-08-d02b-d.pdf", sources: []source{source{url: "https://www.pdp8online.com/pdp8cgi/query_docs/tifftopdf.pl/pdp8docs/maindec-08-d02b-d.pdf", size: 1887293}}},
	fileDesc{filename: "maindec-08-d03a-d.pdf", sources: []source{source{url: "https://svn.so-much-stuff.com/svn/trunk/pdp8/src/dec/maindec-08-d03/maindec-08-d03a-d.pdf", size: 8016849}}},
	fileDesc{filename: "maindec-08-d04b-d.pdf", sources: []source{source{url: "https://deramp.com/downloads/mfe_archive/011-Digital%20Equipment%20Corporation/01%20DEC%20PDP-8%20Family%20Software/03%20MAINDEC%20Maintenance%20progams/MAINDEC%2008/MAINDEC-08%20D04B%20Random%20Jump%20Test/maindec-08-d04b-d.pdf", size: 577733}}},
	fileDesc{filename: "maindec-08-d05b-d.pdf", sources: []source{source{url: "https://deramp.com/downloads/mfe_archive/011-Digital%20Equipment%20Corporation/01%20DEC%20PDP-8%20Family%20Software/03%20MAINDEC%20Maintenance%20progams/MAINDEC%2008/MAINDEC-08%20D05B%20Random%20JMP%20JMS%20Test/maindec-08-d05b-d.pdf", size: 835605}}},
	fileDesc{filename: "maindec-08-d07b-d.pdf", sources: []source{source{url: "https://deramp.com/downloads/mfe_archive/011-Digital%20Equipment%20Corporation/01%20DEC%20PDP-8%20Family%20Software/03%20MAINDEC%20Maintenance%20progams/MAINDEC%2008/MAINDEC-08%20D07B%20Random%20ISZ%20Test/maindec-08-d07b-d.pdf", size: 792817}}},
	fileDesc{filename: "maindec-08-d2ba-d.pdf", sources: []source{source{url: "https://svn.so-much-stuff.com/svn/trunk/pdp8/src/dec/maindec-08-d2b/maindec-08-d2ba-d.pdf", size: 511108}}},
	fileDesc{filename: "maindec-08-d2pe-d.pdf", sources: []source{source{url: "http://www.pdp8online.com/pdp8cgi/query_docs/tifftopdf.pl/pdp8docs/maindec-08-d2pe-d.pdf", size: 2691661}, source{url: "https://deramp.com/downloads/mfe_archive/011-Digital%20Equipment%20Corporation/01%20DEC%20PDP-8%20Family%20Software/03%20MAINDEC%20Maintenance%20progams/MAINDEC%2008/MAINDEC-08%20D2PE%20ASR33%20ASR35%20Test%20Family%20Part%201/maindec-08-d2pe-d.pdf", size: 2691707}}},
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
