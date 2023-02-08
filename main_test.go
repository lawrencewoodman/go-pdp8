package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	missingDecFiles := checkDecFiles()
	missingMaindecFiles := checkMaindecFiles()
	if missingDecFiles || missingMaindecFiles {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func checkDecFiles() bool {
	expectedDecFiles := []string{
		"dec-08-lbaa-pm.bin",
	}
	missing := false
	for _, mf := range expectedDecFiles {
		fullMf := filepath.Join("fixtures", mf)
		if _, err := os.Stat(fullMf); err != nil {
			if !missing {
				missing = true
				fmt.Fprintln(os.Stderr, "Missing DEC diagnostic files.")
				fmt.Fprintf(os.Stderr, "See how to obtain them in %s\r\n",
					filepath.Join("fixtures", "README.md"))
			}
			fmt.Fprintf(os.Stderr, "File missing: %s\n", fullMf)
		}
	}
	return missing
}

func checkMaindecFiles() bool {
	expectedMaindecFiles := []string{
		"maindec-08-d01a-pb.bin",
		"maindec-08-d02b-pb.bin",
	}
	missing := false
	for _, mf := range expectedMaindecFiles {
		fullMf := filepath.Join("fixtures", mf)
		if _, err := os.Stat(fullMf); err != nil {
			if !missing {
				missing = true
				fmt.Fprintln(os.Stderr, "Missing MAINDEC diagnostic files.")
				fmt.Fprintf(os.Stderr, "See how to obtain them in %s\r\n",
					filepath.Join("fixtures", "README.md"))
			}
			fmt.Fprintf(os.Stderr, "File missing: %s\n", fullMf)
		}
	}
	return missing
}
