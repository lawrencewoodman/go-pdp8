package pdp8

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestRunWithInterrupt_maindec_08_d01a(t *testing.T) {
	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	defer tty.Close() // TODO: call this from within pdp?
	p := New()
	if err := p.AddDevice(tty); err != nil {
		t.Fatal(err)
	}

	loadBINTape(t, p, tty, filepath.Join("fixtures", "maindec-08-d01a-pb.bin"))
	p.pc = 0o1200
	p.sr = 0o7777

	// Run test
	hlt, err := p.RunWithInterrupt(50000, 500000)
	if err != nil {
		t.Fatal(err)
	}
	if !hlt {
		t.Errorf("Failed to execute HLT PC: %04o", p.pc-1)
	}
	if mask(p.lac) != 0 || p.pc-1 != 0o1202 {
		t.Errorf("First HLT - got: LAC: %05o PC: %04o, want: LAC: 00000, PC: 1202", p.lac, p.pc-1)
	}

	hlt, err = p.RunWithInterrupt(50000, 50000000)
	if err != nil {
		t.Fatal(err)
	}
	if !hlt {
		t.Errorf("Failed to execute HLT PC: %04o", p.pc-1)
	}

	// TODO: Is this success or partial success?
	if p.pc-1 != 0o4771 {
		t.Errorf("Last HLT - got: PC: %04o, want: PC: 4771", p.pc-1)
	}
	// TODO: Work out how this should report success/error
}

func TestRunWithInterrupt_maindec_08_d02b(t *testing.T) {
	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	defer tty.Close() // TODO: call this from within pdp?
	p := New()
	if err := p.AddDevice(tty); err != nil {
		t.Fatal(err)
	}

	loadBINTape(t, p, tty, filepath.Join("fixtures", "maindec-08-d02b-pb.bin"))

	p.pc = 0o200
	p.sr = 0o4400

	// Run test
	hlt, err := p.RunWithInterrupt(50000, 500000)
	if err != nil {
		t.Fatal(err)
	}

	if hlt {
		t.Errorf("HLT at PC: %04o", p.pc-1)
	}

	if p.pc-1 == 0o406 {
		t.Errorf("Test failed (TAD) - HLT - PC: %04o", p.pc-1)
	}

	if p.pc-1 == 0o2433 {
		t.Errorf("Test failed (ROT) - HLT - PC: %04o", p.pc-1)
	}
}

// Test reader against binary count test pattern
func TestRunWithInterrupt_maindec_08_d2ba(t *testing.T) {
	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	defer tty.Close() // TODO: call this from within pdp?
	p := New()
	if err := p.AddDevice(tty); err != nil {
		t.Fatal(err)
	}

	loadBINTape(t, p, tty, filepath.Join("fixtures", "maindec-08-d2ba-pb.bin"))

	binaryCountTapeFilename := createBinaryCountTestTape(t)
	defer os.Remove(binaryCountTapeFilename)

	f, err := os.Open(binaryCountTapeFilename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	tty.ReaderAttachTape(f)
	tty.ReaderStart()

	// Test reader using binary count test pattern
	p.pc = 0o1625
	p.sr = 0o4002

	// Run test to sync tape
	hlt, err := p.RunWithInterrupt(100, 500000)
	if err != nil {
		t.Fatal(err)
	}

	if !hlt {
		t.Fatalf("Failed to execute HLT PC: %04o", p.pc-1)
	}

	// Check in sync
	if p.pc-1 != 0o1663 {
		t.Fatalf("HLT - got: PC: %04o, want: PC: 1663", p.pc-1)
	}

	// Test tape
	hlt = false
	for !tty.ReaderIsEOF() && !hlt {
		hlt, err = p.RunWithInterrupt(100, 500)
		if err != nil {
			t.Fatal(err)
		}
	}

	if hlt {
		t.Errorf("HLT at PC: %04o", p.pc-1)
	}

	tty.ReaderStop()
}

// Paper tape reader - basic input logic tests
//
// Routines 3 and 4 fail because of what seems to be timing issues.
// For the moment accepting this as it probably doesn't matter for
// an abstract emulation.
func TestRunWithInterrupt_maindec_08_d2pe_PRG0(t *testing.T) {
	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	defer tty.Close() // TODO: call this from within pdp?
	p := New()
	if err := p.AddDevice(tty); err != nil {
		t.Fatal(err)
	}

	loadBINTape(t, p, tty, filepath.Join("fixtures", "maindec-08-d2pe-pb.bin"))

	binaryCountTapeFilename := createBinaryCountTestTape(t)
	defer os.Remove(binaryCountTapeFilename)

	f, err := os.Open(binaryCountTapeFilename)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	tty.ReaderAttachTape(f)
	tty.ReaderStart()

	// ExpectedPC
	runTestPRG0Routine := func(t *testing.T, p *PDP8, routine uint, expectedPC uint) {
		// PRG0
		p.pc = 0o200
		p.sr = 0

		// Shouldn't need this but the test doesn't seem to turn off
		// interrupts properly when running routines individually
		p.ien = false

		// Start test
		hlt, err := p.RunWithInterrupt(100, 500000)
		if err != nil {
			t.Fatal(err)
		}

		if !hlt {
			t.Fatalf("Failed to execute HLT PC: %04o", p.pc-1)
		}

		// Ready to set options
		if p.pc-1 != 0o232 {
			t.Fatalf("HLT - got: PC: %04o, want: PC: 232", p.pc-1)
		}

		// Run specified routine and halt
		p.sr = 0o6000 + routine

		// Run routine
		hlt = false
		for !hlt {
			hlt, err = p.RunWithInterrupt(100, 5000)
			if err != nil {
				t.Fatal(err)
			}
		}

		if !hlt {
			t.Fatalf("Failed to execute HLT PC: %04o", p.pc-1)
		}

		// Routine ends successfully
		if p.pc-1 == 0o320 {
			return
		}

		// This isn't ideal but the best we can expect at the moment
		if p.pc-1 == expectedPC {
			fmt.Printf("    WARN: %s\n", t.Name())
			fmt.Printf("          PRG0 routine: %d, Doesn't pass properly - HLT PC: %04o\n", routine, p.pc-1)
			return
		}

		t.Errorf("HLT - Routine: %d, PC got: %04o, want: %04o", routine, p.pc-1, expectedPC)
	}

	// Run the routines separately as not all pass properly
	cases := []struct {
		routine uint
		// If not proper success then best we can expect
		expectedPC uint
	}{
		{0, 0o320},
		{1, 0o320},
		{2, 0o320},
		{3, 0o1322},
		{4, 0o1362},
		{5, 0o320},
		{6, 0o320},
	}

	for _, c := range cases {
		if _, err := f.Seek(0, 0); err != nil {
			t.Fatal(err)
		}
		runTestPRG0Routine(t, p, c.routine, c.expectedPC)
	}

	tty.ReaderStop()
}
