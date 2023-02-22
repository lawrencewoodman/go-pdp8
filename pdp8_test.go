package pdp8

import (
	"path/filepath"
	"testing"
)

func TestRunWithInterrupt_maindec_08_d01a_pb(t *testing.T) {
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

	// Run maindec tape
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

func TestRunWithInterrupt_maindec_08_d02b_pb(t *testing.T) {
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

	// Run maindec tape
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
