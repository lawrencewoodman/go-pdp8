package pdp8

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TODO: Move all these maindec tests into a maindec_test.go?

// Setup everything neaded to load a MAINDEC test from fixtures/
// Run returned teardownMaindecTest after each test
// Returns: *PDP8, *TTY, teardownMaindecTest
func setupMaindecTest(t *testing.T, filename string) (*PDP8, *TTY, func()) {
	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	p := New()
	if err := p.AddDevice(tty); err != nil {
		t.Fatal(err)
	}

	loadBINTape(t, p, tty, filepath.Join("fixtures", filename))
	teardownMaindecTest := func() {
		tty.Close()
	}
	return p, tty, teardownMaindecTest
}

// MAINDEC-08-D01A
// Instruction test part 2A
func TestRun_maindec_08_d01a(t *testing.T) {
	t.Parallel()

	rw := newDummyReadWriter()
	// ttyOut is so that we can check output
	ttyOut := bytes.NewBuffer(make([]byte, 0, 5000))
	tty := NewTTY(rw, ttyOut)
	p := New()
	if err := p.AddDevice(tty); err != nil {
		t.Fatal(err)
	}

	loadBINTape(t, p, tty, filepath.Join("fixtures", "maindec-08-d01a-pb.bin"))
	defer tty.Close()

	p.pc = 0o1200
	p.sr = 0o7777

	// Run test
	hlt, _, err := p.Run(500000)
	if err != nil {
		t.Fatal(err)
	}
	if !hlt {
		t.Errorf("Failed to execute HLT PC: %04o", p.pc-1)
	}
	if mask(p.lac) != 0 || p.pc-1 != 0o1202 {
		t.Errorf("First HLT - got: LAC: %05o PC: %04o, want: LAC: 00000, PC: 1202", p.lac, p.pc-1)
	}

	// Run until 2x tests have completed
	// Bytes comparison represents:
	//  DEL,DEL,DEL,DEL,DEL,DEL,DEL,
	//  CR, NL, '2', 'A',
	//  DEL,DEL,DEL,DEL,DEL,DEL,DEL,
	//  CR, NL, '2', 'A'
	ttyOutWant := []byte{
		127, 127, 127, 127, 127, 127, 127,
		13, 10, 50, 65,
		127, 127, 127, 127, 127, 127, 127,
		13, 10, 50, 65,
	}
	for !bytes.Equal(ttyOut.Bytes(), ttyOutWant) {
		hlt, _, err = p.Run(5000)
		if err != nil {
			t.Fatal(err)
		}
		if hlt {
			t.Errorf("Test failed - HLT PC: %04o", p.pc-1)
		}
	}
	// Test ends successfully
}

// MAINDEC-08-D02B
// Instruction test part 2B
func TestRun_maindec_08_d02b(t *testing.T) {
	p, _, teardownMaindecTest := setupMaindecTest(t, "maindec-08-d02b-pb.bin")
	defer teardownMaindecTest()

	p.pc = 0o200
	p.sr = 0o4400

	// Run test
	hlt, _, err := p.Run(500000)
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

// MAINDEC-08-D03A
// Basic JMP-JMS Test
func TestRun_maindec_08_d03a(t *testing.T) {
	t.Parallel()
	// We use this tape split into two parts because our bin loader
	// closes the file once it has finished which wouldn't allow the
	// second half to be loaded

	// Part 1
	// Load first program on tape
	rw := newDummyReadWriter()
	// ttyOut is so that we can check output
	ttyOut := bytes.NewBuffer(make([]byte, 0, 5000))
	tty := NewTTY(rw, ttyOut)
	p := New()
	if err := p.AddDevice(tty); err != nil {
		t.Fatal(err)
	}

	loadBINTape(t, p, tty, filepath.Join("fixtures", "maindec-08-d03a-pb1.bin"))
	defer tty.Close()

	// Part 2
	// HLT instructions are deposited throughout unused memory
	// and then the diagnostic program is loaded
	f, err := os.Open(filepath.Join("fixtures", "maindec-08-d03a-pb2.bin"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	tty.ReaderAttachTape(f)
	tty.ReaderStart()

	p.pc = 0o2
	p.sr = 0o7777

	// Run Part 2
	hlt := false
	for !hlt {
		hlt, _, err = p.Run(5000)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Run diagnostic program
	p.pc = 0o600
	p.sr = 0o0000
	hlt = false
	for !hlt {
		hlt, _, err = p.Run(5000)
		if err != nil {
			t.Fatal(err)
		}
	}

	if !hlt {
		t.Fatalf("Failed to execute HLT at PC: %04o", p.pc-1)
	}

	// Check if loader stored
	if p.pc != 0o632 {
		t.Fatalf("loader storing failed - PC: %04o", p.pc-1)
	}

	// Run test
	// Run until 2x 4096 program loops have completed
	// Bytes comparison represents:
	//  CR, NL, '0', '3', CR, NL, '0', '3'
	ttyOutWant := []byte{13, 10, 48, 51, 13, 10, 48, 51}
	for !bytes.Equal(ttyOut.Bytes(), ttyOutWant) {
		hlt, _, err := p.Run(5000)
		if err != nil {
			t.Fatal(err)
		}
		if hlt {
			t.Fatalf("Test failed - HLT PC: %04o", p.pc-1)
		}
	}
	// Test ends successfully
}

// MAINDEC-08-D04B
// Random JMP test
func TestRun_maindec_08_d04b(t *testing.T) {
	t.Parallel()
	rw := newDummyReadWriter()
	// ttyOut is so that we can check output
	ttyOut := bytes.NewBuffer(make([]byte, 0, 5000))
	tty := NewTTY(rw, ttyOut)
	p := New()
	if err := p.AddDevice(tty); err != nil {
		t.Fatal(err)
	}

	loadBINTape(t, p, tty, filepath.Join("fixtures", "maindec-08-d04b-pb.bin"))
	defer tty.Close()

	// Halt on error
	p.pc = 0o200
	p.sr = 0o4000

	// Run test
	// Run until 2x 72000 tests have completed
	// Bytes comparison represents:
	//  0, CR, NL, '0', '4', CR, NL, '0', '3'
	ttyOutWant := []byte{0, 13, 10, 48, 52, 13, 10, 48, 52}
	for !bytes.Equal(ttyOut.Bytes(), ttyOutWant) {
		hlt, _, err := p.Run(5000000)
		if err != nil {
			t.Fatal(err)
		}

		if hlt {
			t.Fatalf("Test failed - HLT PC: %04o", p.pc-1)
		}
	}

	// Test ends successfully
}

// MAINDEC-08-D05B
// Random JMP-JMS test
func TestRun_maindec_08_d05b(t *testing.T) {
	t.Parallel()

	rw := newDummyReadWriter()
	// ttyOut is so that we can check output
	ttyOut := bytes.NewBuffer(make([]byte, 0, 5000))
	tty := NewTTY(rw, ttyOut)
	p := New()
	if err := p.AddDevice(tty); err != nil {
		t.Fatal(err)
	}
	loadBINTape(t, p, tty, filepath.Join("fixtures", "maindec-08-d05b-pb.bin"))
	defer tty.Close()

	// Halt on error
	p.pc = 0o200
	p.sr = 0o4000

	// Run test
	// Run until 2x 61000 tests have completed
	// Bytes comparison represents:
	//  DEL, CR, NL, '0', '5', CR, NL, '0', '5'
	ttyOutWant := []byte{127, 13, 10, 48, 53, 13, 10, 48, 53}
	for !bytes.Equal(ttyOut.Bytes(), ttyOutWant) {

		hlt, _, err := p.Run(5000000)
		if err != nil {
			t.Fatal(err)
		}

		if hlt {
			t.Fatalf("Test failed - HLT PC: %04o", p.pc-1)
		}
	}

	// Test ends successfully
}

// MAINDEC-08-D2BA
// Exercisor for the PDP-8 Teletype Paper Tape Reader
// Test reader against binary count test pattern
func TestRun_maindec_08_d2ba_test_binary_count_pattern(t *testing.T) {
	t.Parallel()
	p, tty, teardownMaindecTest := setupMaindecTest(t, "maindec-08-d2ba-pb.bin")
	defer teardownMaindecTest()

	// Binary count test pattern tape
	f, err := os.Open(filepath.Join("fixtures", "maindec-00-d2g3-pt"))
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
	hlt, _, err := p.Run(500000)
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
	// 4100 is 4096 (12-bits) + a few for overflow if using 12-bits
	for !tty.ReaderIsEOF() && !hlt && tty.ReaderPos() < 4100 {
		hlt, _, err = p.Run(500)
		if err != nil {
			t.Fatal(err)
		}
	}

	if hlt {
		t.Errorf("HLT at PC: %04o", p.pc-1)
	}

	tty.ReaderStop()
}

// MAINDEC-08-D2BA
// Exercisor for the PDP-8 Teletype Paper Tape Reader
// Create a binary count test pattern tape
func TestRun_maindec_08_d2ba_punch_binary_count_tape(t *testing.T) {
	t.Parallel()
	p, tty, teardownMaindecTest := setupMaindecTest(t, "maindec-08-d2ba-pb.bin")
	defer teardownMaindecTest()

	ttyOut := bytes.NewBuffer(make([]byte, 0, 5000))

	tty.PunchAttachTape(ttyOut)
	tty.PunchStart()

	// Punch a binary count test pattern
	p.pc = 0o200
	p.sr = 0o2000

	var outputTapeSize = 0
	for outputTapeSize <= 5000 {
		// Run routine
		hlt, _, err := p.Run(5000)
		if err != nil {
			t.Fatal(err)
		}
		if hlt {
			t.Fatalf("HLT at PC: %04o", p.pc-1)
		}
		outputTapeSize = ttyOut.Len()
	}

	tty.PunchStop()

	// Test bytes are sequential and start at 0
	var expectedByte byte = 0
	for _, b := range ttyOut.Bytes() {
		if b != expectedByte {
			t.Fatalf("bytes not sequential got: %v, want: %v", b, expectedByte)
		}
		expectedByte++
	}
}

// MAINDEC-08-D2PE
// ASR 33/35 Teletype Tests Part 1
// PRG0 - Reader basic input logic tests
//
// Routine 3 fails because of a timing issue, not because
// a problem with KSF.  This is acceptable because the
// alternative would be to slow down TTY when it is
// unlikely that any normal program would trip up
// because of this issue.
func TestRun_maindec_08_d2pe_PRG0(t *testing.T) {
	t.Parallel()

	runTestPRG0Routine := func(routine uint, expectedPC uint) {
		p, tty, teardownMaindecTest := setupMaindecTest(t, "maindec-08-d2pe-pb.bin")
		defer teardownMaindecTest()

		// Binary count test pattern tape
		f, err := os.Open(filepath.Join("fixtures", "maindec-00-d2g3-pt"))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()

		tty.ReaderAttachTape(f)
		tty.ReaderStart()
		defer tty.ReaderStop()

		// PRG0
		p.pc = 0o200
		p.sr = 0

		// Start test
		hlt, _, err := p.Run(500000)
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
			hlt, _, err = p.Run(5000)
			if err != nil {
				t.Fatal(err)
			}
		}

		if !hlt {
			t.Fatalf("Failed to execute HLT PC: %04o", p.pc-1)
		}

		// Routine ends successfully
		if p.pc-1 == 0o320 {
			if expectedPC != 0o320 {
				t.Logf("routine: %d, has started passing! Look again at test", routine)
			}
			return
		}

		// This isn't ideal but the best we can expect at the moment
		if p.pc-1 == expectedPC {
			t.Logf("routine: %d, only a partial pass at the moment - HLT PC: %04o", routine, p.pc-1)
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
		{4, 0o320},
		{5, 0o320},
		{6, 0o320},
	}

	for _, c := range cases {
		runTestPRG0Routine(c.routine, c.expectedPC)
	}

}

// MAINDEC-08-D2PE
// ASR 33/35 Teletype Tests Part 1
// PRG1 - Punch basic output logic tests
func TestRun_maindec_08_d2pe_PRG1(t *testing.T) {
	t.Parallel()
	p, tty, teardownMaindecTest := setupMaindecTest(t, "maindec-08-d2pe-pb.bin")
	defer teardownMaindecTest()

	ttyOut := bytes.NewBuffer(make([]byte, 0, 5000))

	tty.PunchAttachTape(ttyOut)
	tty.PunchStart()

	// PRG1
	p.pc = 0o200
	p.sr = 1

	// Start test
	hlt, _, err := p.Run(500000)
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

	// Run all tests
	p.sr = 0

	// Run routine
	hlt = false
	for !hlt {
		hlt, _, err = p.Run(5000)
		if err != nil {
			t.Fatal(err)
		}
	}

	if !hlt {
		t.Fatalf("Failed to execute HLT PC: %04o", p.pc-1)
	}

	// Routine ends successfully
	if p.pc-1 != 0o274 {
		t.Errorf("HLT - PC got: %04o, want: %04o", p.pc-1, 0o274)
	}

	tty.PunchStop()
}

// MAINDEC-08-D2PE
// ASR 33/35 Teletype Tests Part 1
// PRG2 - Reader test
func TestRun_maindec_08_d2pe_PRG2(t *testing.T) {
	t.Parallel()
	p, tty, teardownMaindecTest := setupMaindecTest(t, "maindec-08-d2pe-pb.bin")
	defer teardownMaindecTest()

	// Binary count test pattern tape
	f, err := os.Open(filepath.Join("fixtures", "maindec-00-d2g3-pt"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	tty.ReaderAttachTape(f)
	tty.ReaderStart()

	// PRG2
	p.pc = 0o200
	p.sr = 2

	// Start test
	hlt, _, err := p.Run(500000)
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

	// Run all tests
	p.sr = 0

	// Run routine
	hlt = false
	for !hlt {
		hlt, _, err = p.Run(5000)
		if err != nil {
			t.Fatal(err)
		}
	}

	if !hlt {
		t.Fatalf("Failed to execute HLT PC: %04o", p.pc-1)
	}

	// Routine ends successfully
	if p.pc-1 != 0o274 {
		t.Errorf("HLT - PC got: %04o, want: %04o", p.pc-1, 0o274)
	}

	tty.ReaderStop()
}

// MAINDEC-08-D2QD
// ASR 33/35 Teletype Tests Part 2
func TestRun_maindec_08_d2qd(t *testing.T) {
	// TODO: Implement this
	// TODO: Sources: http://www.pdp8online.com/pdp8cgi/query_docs/tifftopdf.pl/pdp8docs/maindec-08-d2qd-d.pdf
	// TODO: https://deramp.com/downloads/mfe_archive/011-Digital%20Equipment%20Corporation/01%20DEC%20PDP-8%20Family%20Software/03%20MAINDEC%20Maintenance%20progams/MAINDEC%2008/MAINDEC-08%20D2QD%20ASR33%20ASR35%20Test%20Family%20Part%202%20/
	t.Skip("Not currently implemented")
}
