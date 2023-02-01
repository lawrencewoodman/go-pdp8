/*
 * A PDP-8 emulator
 *
 * Copyright (C) 2023 Lawrence Woodman <lwoodman@vlifesystems.com>
 *
 * Licensed under an MIT licence.  Please see LICENCE.md for details.
 */

package main

// TODO: Work out why \r\n is required in print strings
import (
	"fmt"
	"io"
	"os"
	"strconv"
)

const memSize = 4096

type pdp8 struct {
	// NOTE: Using uint rather than int because of right shifting
	// TODO: consider creating a word type to better encapsulate this?
	mem           [memSize]uint // Memory
	pc            uint          // Program counter
	ir            uint          // Instruction register
	sr            uint          // Switch register
	lac           uint          // Accumulator register 13th bit is Link flag
	mq            uint          // Multiplier Quotient
	ien           bool          // Whether interrupts are enabled
	devices       []device      // Devices for IOT
	deviceNumbers []int         // The device numbers currently registered
}

// TODO: Put this in a separate package as New
func newPdp8() *pdp8 {
	p := &pdp8{}
	p.pc = 0o200
	p.sr = 0
	p.lac = 0
	return p
}

// Returns the lower 12-bits
func mask(w uint) uint {
	return w & 0o7777
}

// TODO: Consider putting link bit in p.l
// Returns the lower 13-bits i.e. includes link bit
func lmask(w uint) uint {
	return w & 0o17777
}

// TODO: Decide if to use this
func printPunchHoles(n uint) {
	if n > 255 {
		panic(fmt.Sprintf("punch num too big: %d", n))
	}
	fmt.Printf("%05b %03b\r\n", (n&0o370)>>3, n&0o7)
}

func (p *pdp8) load(filename string) error {
	var n int
	var c uint
	var addr uint
	// NOTE: The checksum is the sum of each byte of data
	// NOTE: NOT each word
	var checksum uint
	b := make([]byte, 1)

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Skip until run-in found
	for {
		n, err = f.Read(b)
		c := uint(b[0])
		if n == 0 || err == io.EOF || c == 0o200 {
			break
		}
	}
	if err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	for {
		_, err = f.Read(b)
		if err != nil {
			return err
		}
		c = uint(b[0])

		// Skip run-in
		if (c & 0o200) != 0 {
			continue
		}

		hi := c << 6 // High 6 bits
		_, err = f.Read(b)
		if err != nil {
			return err
		}
		c = uint(b[0])
		c = hi | (c & 0o77) // Make 12-bit word

		// Look for run-out, to ignore word before it as being a checksum
		_, err = f.Read(b)
		if err != nil {
			return err
		}
		d := uint(b[0])

		// If run-out word
		if (d & 0o200) != 0 {
			break
		}

		// Not run-out word so unget char
		_, err = f.Seek(-1, 1)
		if err != nil {
			return err
		}

		// Process word
		// If 13th bit set, the word specifies an address
		// Else it is a word to put at the current address
		if (c & 0o10000) != 0 {
			if addr != 0 {
				fmt.Printf("-%04o", addr-1)
			}
			addr = mask(c)
			fmt.Printf(" %04o", addr)
		} else {
			p.mem[addr] = c
			checksum = mask(checksum + c&0o77)
			checksum = mask(checksum + (c & 0o7700 >> 6))
			addr = addr + 1
		}
	}

	fmt.Printf("-%04o\r\n", mask(addr-1))
	fmt.Printf(" CHECKSUM ")
	if checksum == mask(c) {
		fmt.Printf("OK: %04o\r\n", checksum)
	} else {
		fmt.Printf("FAIL: %04o, SHOULD BE: %04o\r\n", checksum, mask(c))
		//		os.Exit(1)
		// TODO: What to do if fails?
	}
	return nil
}

func (p *pdp8) regDevice(d device) error {
	newDeviceNumbers := d.deviceNumbers()
	for _, n1 := range newDeviceNumbers {
		for _, n2 := range p.deviceNumbers {
			if n1 == n2 {
				return fmt.Errorf("device number conflict: %02o", n1)
			}
		}
		p.deviceNumbers = append(p.deviceNumbers, n1)
	}
	p.devices = append(p.devices, d)
	return nil
}

func usage(errMsg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\r\n", errMsg)
	fmt.Fprintf(os.Stderr, "Usage: %s binrimfile ?pc? ?sr? ?-v?\r\n", os.Args[0])
}

func main() {
	// TODO: Change this order and usage
	if len(os.Args) < 2 {
		usage("no filename supplied")
		os.Exit(1)
	}

	_tty, err := newTty()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\r\n", err)
		os.Exit(1)
	}
	defer _tty.close() // TODO: call this from within pdp?
	p := newPdp8()
	if err := p.regDevice(_tty); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\r\n", err)
		os.Exit(1)
	}

	if err := p.load(os.Args[1]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\r\n", err)
		os.Exit(1)
	}

	if len(os.Args) >= 3 {
		pc, err := strconv.ParseInt(os.Args[2], 8, 0)
		if err != nil {
			usage(fmt.Sprintf("invalid argument: %s", os.Args[2]))
			os.Exit(1)
		}
		p.pc = uint(pc)
	}

	if len(os.Args) >= 4 {
		sr, err := strconv.ParseInt(os.Args[3], 8, 0)
		if err != nil {
			usage(fmt.Sprintf("invalid argument: %s", os.Args[3]))
			os.Exit(1)
		}
		p.sr = uint(sr)
	}
	defer cleanup(p)
	if err := p.runWithInterrupt(50000); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\r\n", err)
		os.Exit(1)
	}

	fmt.Printf(" PC: %04o, LAC: %05o\r\n", mask(p.pc-1), p.lac)
}

func (p *pdp8) runWithInterrupt(cyclesPerInterrupt int) error {
	var err error
	var hlt bool

	for {
		hlt, err = p.run(cyclesPerInterrupt)
		if err != nil {
			return err
		}

		if hlt {
			// HLT before interrupt otherwise PC will move
			fmt.Printf(" HLT  PC: %04o, LAC: %05o\r\n", mask(p.pc-1), p.lac)
			break
		}

		// Poll for interrupts if interrupts enabled
		if p.ien {
			for _, d := range p.devices {
				if d.interrupt() {
					p.mem[0] = p.pc
					p.pc = 1
					p.ien = false
					break
				}
			}
		}
	}
	return err
}

// TODO: rename this
func cleanup(p *pdp8) {
	fmt.Printf(" PC %04o\r\n", mask(p.pc-1))
}

// fetch returns opCode and opAddr if relevant else 0
func (p *pdp8) fetch() (opCode uint, opAddr uint) {
	p.ir = p.mem[p.pc]
	opCode = (p.ir >> 9) & 0o7
	opAddr = 0

	if opCode <= 5 { // If <= JMP and hence includes an address
		opAddr = p.ir & 0o177
		if (p.ir & 0o200) != 0 { // If zero page
			opAddr |= p.pc & 0o7600
		}

		// If indirect
		if (p.ir & 0o400) != 0 {
			// If auto increment address
			if (opAddr & 0o7770) == 0o10 {
				p.mem[opAddr] = mask(p.mem[opAddr] + 1)
			}
			opAddr = p.mem[opAddr]
		}
	}

	// TODO: This is wrong because -v could be passed earlier without
	// TODO: pc or sr
	if len(os.Args) >= 5 {
		fmt.Printf("PC: %04o  IR:  %04o  LAC: %05o\r\n", p.pc, p.ir, p.lac)
	}

	p.pc = mask(p.pc + 1)
	return opCode, opAddr
}

// Returns (hltExecuted, error)
// TODO: Improve cycle accuracy and return number left/over?
func (p *pdp8) run(cycles int) (bool, error) {
	var err error
	var hlt bool

	for cycles > 0 {
		opCode, opAddr := p.fetch()
		hlt, err = p.execute(opCode, opAddr)
		if err != nil || hlt {
			break
		}
		cycles -= 1
	}
	return hlt, err
}

// Returns (hltExecuted, error)
func (p *pdp8) execute(opCode uint, opAddr uint) (bool, error) {
	var err error
	var hlt bool

	switch opCode {
	case 0: // AND
		p.lac &= p.mem[opAddr] | 0o10000
	case 1: // TAD
		p.lac = lmask(p.lac + p.mem[opAddr])
	case 2: // ISZ
		p.mem[opAddr] = mask(p.mem[opAddr] + 1)
		if p.mem[opAddr] == 0 {
			p.pc = mask(p.pc + 1)
		}
	case 3: // DCA
		p.mem[opAddr] = mask(p.lac)
		p.lac &= 0o10000
	case 4: // JMS
		p.mem[opAddr] = p.pc
		p.pc = mask(opAddr + 1)
	case 5: // JMP
		p.pc = opAddr
	case 6: // IOT
		err = p.iot()
	case 7: // OPR
		hlt = p.opr()
	}
	return hlt, err
}

// IOT instruction
func (p *pdp8) iot() error {
	var err error
	device := (p.ir >> 3) & 0o77
	iotOp := p.ir & 0o7
	switch device {
	case 0o0: // CPU
		switch iotOp {
		case 0o1: // ION
			p.ien = true
		case 0o2: // IOF
			p.ien = false
		default:
			// TODO: Report an unknown op?
		}
	default:
		for _, d := range p.devices {
			p.pc, p.lac, err = d.iot(p.ir, p.pc, p.lac)
			if err != nil {
				return err
				// TODO: add context
			}
		}
	}
	return err
}

// OPR instruction (microcoded instructions)
// Returns whether HLT (Halt) has been executed
func (p *pdp8) opr() bool {
	// TODO: Check order as well as AND/OR combinations
	if (p.ir & 0o400) == 0 { // Group 1
		if (p.ir & 0o200) != 0 { // CLA
			p.lac = p.lac & 0o10000
		}
		if (p.ir & 0o100) != 0 { // CLL
			p.lac = p.lac & 0o7777
		}
		if (p.ir & 0o40) != 0 { // CMA
			p.lac = p.lac ^ 0o7777
		}
		if (p.ir & 0o20) != 0 { // CML
			p.lac = p.lac ^ 0o10000
		}
		if (p.ir & 0o1) != 0 { // IAC
			p.lac = lmask(p.lac + 1)
		}
		switch p.ir & 0o16 {
		case 0o12: // RTR
			p.lac = lmask((p.lac >> 1) | (p.lac << 12))
			p.lac = lmask((p.lac >> 1) | (p.lac << 12))
		case 0o10: // RAR
			p.lac = lmask((p.lac >> 1) | (p.lac << 12))
		case 0o6: // RTL
			p.lac = lmask((p.lac >> 12) | (p.lac << 1))
			p.lac = lmask((p.lac >> 12) | (p.lac << 1))
		case 0o4: // RAL
			p.lac = lmask((p.lac >> 12) | (p.lac << 1))
		case 0o2: // BSW
			// TODO: Should this be able to be called with
			// TODO: one of: RTR, RAR, RTL, RAL
			p.lac = (p.lac & 0o10000) |
				((p.lac >> 6) & 0o77) | ((p.lac << 6) & 0o7700)
		}
	} else if (p.ir & 0o1) == 0 { // Group 2
		var sv uint
		// SMA, SPA, SZA, SNA, SNL, SZL
		// TODO: Split this out to make it clearer
		sc := ((p.ir&0o100) != 0 && (p.lac&0o4000) != 0) ||
			((p.ir&0o40) != 0 && (p.lac&0o7777) == 0) ||
			(p.ir&0o20) != 0 && (p.lac&0o10000) != 0
		if sc {
			sv = 0
		} else {
			sv = 0o10
		}
		if sv == (p.ir & 0o10) {
			p.pc = mask(p.pc + 1)
		}
		if (p.ir & 0o200) != 0 { // CLA
			p.lac &= 0o10000
		}
		if (p.ir & 0o4) != 0 { // OSR
			p.lac |= p.sr
		}
		if (p.ir & 0o2) != 0 { // HLT
			return true
		}
	} else { // Group 3
		// We store MQ so that MQA and MQL can exchange MQ and AC
		t := p.mq
		if (p.ir & 0o201) != 0 { // CLA
			p.lac &= 0o10000
		}
		if (p.ir & 0o21) != 0 { // MQL
			p.mq = p.lac & 0o7777
			p.lac &= 0o10000
		}
		if (p.ir & 0o101) != 0 { // MQA
			p.lac |= t
		}
	}
	return false
}
