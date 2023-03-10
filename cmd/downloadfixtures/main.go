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
	fileDesc{filename: "maindec-08-d01a-pb.bin", sources: []source{source{url: "http://dustyoldcomputers.com/pdp-common/reference/papertapes/maindec/maindec-08-d01a-pb.bin", size: 4328}, source{url: "https://svn.so-much-stuff.com/svn/trunk/pdp8/src/dec/maindec-08-d01/maindec-08-d01a-pb", size: 4328}}},
	fileDesc{filename: "maindec-08-d02b-pb.bin", sources: []source{source{url: "http://dustyoldcomputers.com/pdp-common/reference/papertapes/maindec/maindec-08-d02b-pb.bin", size: 1876}}},
	fileDesc{filename: "maindec-08-d03a-pb1.bin", sources: []source{source{url: "https://svn.so-much-stuff.com/svn/trunk/pdp8/src/dec/maindec-08-d03/maindec-08-d03a-pb1", size: 198}}},
	fileDesc{filename: "maindec-08-d03a-pb2.bin", sources: []source{source{url: "https://svn.so-much-stuff.com/svn/trunk/pdp8/src/dec/maindec-08-d03/maindec-08-d03a-pb2", size: 3464}}},
	fileDesc{filename: "maindec-08-d04b-pb.bin", sources: []source{source{url: "http://www.pdp8online.com/ftp/software/paper_tapes/alltapes/maindec-08-d04b-pb.bin", size: 1011}, source{url: "https://deramp.com/downloads/mfe_archive/011-Digital%20Equipment%20Corporation/01%20DEC%20PDP-8%20Family%20Software/03%20MAINDEC%20Maintenance%20progams/MAINDEC%2008/MAINDEC-08%20D04B%20Random%20Jump%20Test/maindec-08-d04b-pb", size: 692}}},
	fileDesc{filename: "maindec-08-d05b-pb.bin", sources: []source{source{url: "http://www.pdp8online.com/ftp/software/paper_tapes/alltapes/maindec-08-d05b-pb.bin", size: 1243}, source{url: "https://deramp.com/downloads/mfe_archive/011-Digital%20Equipment%20Corporation/01%20DEC%20PDP-8%20Family%20Software/03%20MAINDEC%20Maintenance%20progams/MAINDEC%2008/MAINDEC-08%20D05B%20Random%20JMP%20JMS%20Test/maindec-08-d05b-pb", size: 924}}},
	fileDesc{filename: "maindec-08-d07b-pb.bin", sources: []source{source{url: "http://www.pdp8online.com/ftp/software/paper_tapes/alltapes/maindec-08-d07b-pb.bin", size: 1343}, source{url: "https://deramp.com/downloads/mfe_archive/011-Digital%20Equipment%20Corporation/01%20DEC%20PDP-8%20Family%20Software/03%20MAINDEC%20Maintenance%20progams/MAINDEC%2008/MAINDEC-08%20D07B%20Random%20ISZ%20Test/maindec-08-d07b-pb", size: 1024}}},
	fileDesc{filename: "maindec-08-d2ba-pb.bin", sources: []source{source{url: "https://svn.so-much-stuff.com/svn/trunk/pdp8/src/dec/maindec-08-d2b/maindec-08-d2ba-pb", size: 1277}}},
	fileDesc{filename: "maindec-00-d2g3-pt", sources: []source{source{url: "https://svn.so-much-stuff.com/svn/trunk/pdp8/src/dec/maindec-00-d2g3/maindec-00-d2g3-pt", size: 14698}}},
	fileDesc{filename: "maindec-08-d2pe-pb.bin", sources: []source{source{url: "https://deramp.com/downloads/mfe_archive/011-Digital%20Equipment%20Corporation/01%20DEC%20PDP-8%20Family%20Software/03%20MAINDEC%20Maintenance%20progams/MAINDEC%2008/MAINDEC-08%20D2PE%20ASR33%20ASR35%20Test%20Family%20Part%201/maindec-08-d2pe-pb", size: 2036}, source{url: "https://svn.so-much-stuff.com/svn/trunk/pdp8/src/dec/maindec-08-d2p/maindec-08-d2pe-pb", size: 2036}}},
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
