package main

import (
	"path/filepath"
	"testing"
)

func TestRunWithInterrupt_maindec_08_d01a_pb(t *testing.T) {
	_tty, err := newHeadlessTty()
	if err != nil {
		t.Fatal(err)
	}
	defer _tty.close() // TODO: call this from within pdp?
	p := newPdp8()
	if err := p.regDevice(_tty); err != nil {
		t.Fatal(err)
	}

	loadTape(t, p, _tty, filepath.Join("fixtures", "maindec-08-d01a-pb.bin"))
	p.pc = 0o1200
	p.sr = 0o7777

	// Run maindec tape
	if err := p.runWithInterrupt(50000, 500000); err != nil {
		t.Fatal(err)
	}
	if mask(p.lac) != 0 || p.pc-1 != 0o1202 {
		t.Errorf("First HLT - got: LAC: %05o PC: %04o, want: LAC: 00000, PC: 1202", p.lac, p.pc-1)
	}

	if err := p.runWithInterrupt(50000, 50000000); err != nil {
		t.Fatal(err)
	}

	// TODO: Is this success or partial success?
	if p.pc-1 != 0o4771 {
		t.Errorf("Last HLT - got: PC: %04o, want: PC: 4771", p.pc-1)
	}
	// TODO: Work out how this should report success/error
}

func TestRunWithInterrupt_maindec_08_d02b_pb(t *testing.T) {
	_tty, err := newHeadlessTty()
	if err != nil {
		t.Fatal(err)
	}
	defer _tty.close() // TODO: call this from within pdp?
	p := newPdp8()
	if err := p.regDevice(_tty); err != nil {
		t.Fatal(err)
	}

	loadTape(t, p, _tty, filepath.Join("fixtures", "maindec-08-d02b-pb.bin"))
	p.pc = 0o200
	p.sr = 0o4400

	// Run maindec tape
	if err := p.runWithInterrupt(50000, 500000); err != nil {
		t.Fatal(err)
	}
	if p.pc-1 == 0o406 {
		t.Errorf("Test failed (TAD) - HLT - PC: %04o", p.pc-1)
	}

	if p.pc-1 == 0o2433 {
		t.Errorf("Test failed (ROT) - HLT - PC: %04o", p.pc-1)
	}
}
