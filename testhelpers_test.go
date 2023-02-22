/*
 * Test helper functions
 */

package pdp8

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// Load paper tape in binary format
func loadBINTape(t *testing.T, p *PDP8, tty *TTY, filename string) {
	// Load the BIN loader
	err := p.LoadRIMTape(tty, filepath.Join("fixtures", "dec-08-lbaa-pm.rim"))
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Open(filename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	// Run BIN loader to load supplied paper tape
	tty.ReaderAttachTape(bufio.NewReader(f))

	p.pc = 0o7777

	// A 1 in the MSB of SR indicates the low-speed reader,
	// that is the key / ASR-33
	// A 0 in the MSB of SR indicates a high-speed reader
	p.sr = 0o7777

	// Start the punched tape reader
	tty.ReaderStart()

	var hlt bool = false
	for !tty.ReaderIsEOF() && !hlt {
		// Run binary loader to load paper tape
		hlt, err = p.RunWithInterrupt(1000, 10000)
		if err != nil {
			fmt.Printf("hlt\n")
			t.Fatal(err)
		}
	}

	// Stop the punched tape reader
	tty.ReaderStop()

	// TODO: potentially could finish run at end of tape
	// TODO: before HLT is executed
	// TODO: need to check for this
	if !hlt {
		t.Errorf("Failed to execute HLT at PC: %04o", p.pc-1)
	}

	if mask(p.lac) != 0 || p.ir != 0o7402 {
		t.Fatalf("Checksum fail for tape: %s", filename)
	}
}

// TODO: For debugging - do we need this here?
func dumpMemory(startLocation uint, mem [4096]uint) {
	for n := startLocation; n <= 0o7777; n++ {
		if n%6 == 0 {
			fmt.Printf("\n%04o: ", n)
		}
		fmt.Printf("%04o ", mem[n])
	}
	fmt.Printf("\n")
}

// TODO: See if something like this already exists
type dummyReadWriter struct {
}

func newDummyReadWriter() *dummyReadWriter {
	return &dummyReadWriter{}
}

func (r *dummyReadWriter) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (r *dummyReadWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}
