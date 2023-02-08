/*
 * Test helper functions
 */

package main

import (
	"path/filepath"
	"testing"
)

func loadTape(t *testing.T, p *pdp8, _tty *headlessTty, filename string) {
	if err := p.load(filepath.Join("fixtures", "dec-08-lbaa-pm.bin")); err != nil {
		t.Fatal(err)
	}

	if err := _tty.attachReaderTape(filename); err != nil {
		t.Fatal(err)
	}

	p.pc = 0o7777

	// A minus number in SR indicates reader is from
	// Key / ASR-33 not highspeed reader
	p.sr = 0o7000

	// Run binary loader to load maindec tape
	// TODO: Is this long enough?
	if err := p.runWithInterrupt(50000, 5000000); err != nil {
		t.Fatal(err)
	}

	if mask(p.lac) != 0 || p.ir != 0o7402 {
		t.Fatalf("Checksum fail for tape: %s", filename)
	}
}
