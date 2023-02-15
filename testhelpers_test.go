/*
 * Test helper functions
 */

package main

import (
	"fmt"
	"path/filepath"
	"testing"
)

// Load paper tape in RIM format
func loadRIMTape(t *testing.T, p *pdp8, _tty *headlessTty, filename string) {
	rimLowSpeedLoader := map[uint]uint{
		0o7756: 0o6032,
		0o7757: 0o6031,
		0o7760: 0o5357,
		0o7761: 0o6036,
		0o7762: 0o7106,
		0o7763: 0o7006,
		0o7764: 0o7510,
		0o7765: 0o5357,
		0o7766: 0o7006,
		0o7767: 0o6031,
		0o7770: 0o5367,
		0o7771: 0o6034,
		0o7772: 0o7420,
		0o7773: 0o3776,
		0o7774: 0o3376,
		0o7775: 0o5356,
		0o7776: 0o0,
		0o7777: 0o0,
	}

	for addr, v := range rimLowSpeedLoader {
		p.mem[addr] = v
	}

	// Attach Paper tape in RIM format
	if err := _tty.attachReaderTape(filename); err != nil {
		t.Fatal(err)
	}

	// Start of RIM loader
	p.pc = 0o7756

	cyclesCount := 10000
	for {
		// Run RIM loader to load the paper tape
		if err := p.runWithInterrupt(50000, 50000); err != nil {
			t.Fatal(err)
		}
		cyclesCount--
		if _tty.isEOF() || cyclesCount == 0 {
			break
		}
	}

	if !_tty.isEOF() || !(p.pc == 0o7756 || p.pc == 0o7760) {
		t.Fatalf("RIM loader didn't finish, PC: %04o", p.pc)
	}
}

// Load paper tape in binary format
func loadBINTape(t *testing.T, p *pdp8, _tty *headlessTty, filename string) {
	// Load the BIN loader
	loadRIMTape(t, p, _tty, filepath.Join("fixtures", "dec-08-lbaa.rim"))

	// Run BIN loader to load supplied paper tape
	if err := _tty.attachReaderTape(filename); err != nil {
		t.Fatal(err)
	}

	p.pc = 0o7777

	// A 1 in the MSB of SR indicates the low-speed reader,
	// that is the key / ASR-33
	// A 0 in the MSB of SR indicates a high-speed reader
	p.sr = 0o7777

	// Run binary loader to load maindec tape
	// TODO: Is this long enough?
	if err := p.runWithInterrupt(50000, 5000000); err != nil {
		t.Fatal(err)
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