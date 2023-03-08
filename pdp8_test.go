package pdp8

import (
	"testing"
)

// TSF won't become ready until a TPC or TLS instruction has
// been executed
func TestRun_TSF_not_ready_until_TPC_or_TLS(t *testing.T) {
	const (
		JMP = 0o5200
		TSF = 0o6041
		HLT = 0o7402
	)

	testRoutine := map[uint]uint{
		0o200: TSF,
		0o201: JMP + 0,
		0o202: HLT,
	}

	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	defer tty.Close()

	p := New()
	if err := p.AddDevice(tty); err != nil {
		t.Fatal(err)
	}

	// Load routine
	for addr, v := range testRoutine {
		p.mem[addr] = v
	}

	p.pc = 0o200

	// Run test
	hlt, _, err := p.Run(500)
	if err != nil {
		t.Fatal(err)
	}

	if hlt {
		t.Errorf("HLT PC: %04o", p.pc-1)
	}
}

func TestRun_TSF_ready_after_TPC(t *testing.T) {
	const (
		JMP = 0o5200
		TSF = 0o6041
		TPC = 0o6044
		CLA = 0o7200
		HLT = 0o7402
	)

	testRoutine := map[uint]uint{
		0o200: TSF,
		0o201: JMP + 3,
		0o202: HLT,
		0o203: CLA,
		0o204: TPC,
		0o205: TSF,
		0o206: JMP + 5,
		0o207: HLT,
	}

	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	defer tty.Close()

	p := New()
	if err := p.AddDevice(tty); err != nil {
		t.Fatal(err)
	}

	// Load routine
	for addr, v := range testRoutine {
		p.mem[addr] = v
	}

	p.pc = 0o200

	// Run test
	hlt, _, err := p.Run(500)
	if err != nil {
		t.Fatal(err)
	}

	if !hlt {
		t.Errorf("Failed to execute HLT PC: %04o", p.pc-1)
	}
	if p.pc-1 != 0o207 {
		t.Errorf("got: PC: %04o, want: PC: 207", p.pc-1)
	}
}

func TestRun_TSF_ready_after_TLS(t *testing.T) {
	const (
		JMP = 0o5200
		TSF = 0o6041
		TLS = 0o6046
		CLA = 0o7200
		HLT = 0o7402
	)

	testRoutine := map[uint]uint{
		0o200: TSF,
		0o201: JMP + 3,
		0o202: HLT,
		0o203: CLA,
		0o204: TLS,
		0o205: TSF,
		0o206: JMP + 5,
		0o207: HLT,
	}

	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	defer tty.Close()

	p := New()
	if err := p.AddDevice(tty); err != nil {
		t.Fatal(err)
	}

	// Load routine
	for addr, v := range testRoutine {
		p.mem[addr] = v
	}

	p.pc = 0o200

	// Run test
	hlt, _, err := p.Run(500)
	if err != nil {
		t.Fatal(err)
	}

	if !hlt {
		t.Errorf("Failed to execute HLT PC: %04o", p.pc-1)
	}
	if p.pc-1 != 0o207 {
		t.Errorf("got: PC: %04o, want: PC: 207", p.pc-1)
	}
}
