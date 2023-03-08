/*
 * An embeddable PDP-8 emulator library
 *
 * Copyright (C) 2023 Lawrence Woodman <lwoodman@vlifesystems.com>
 *
 * Licensed under an MIT licence.  Please see LICENCE.md for details.
 */

package pdp8

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

const memSize = 4096

type PDP8 struct {
	// NOTE: Using uint rather than int because of right shifting
	// TODO: consider creating a word type to better encapsulate this?
	mem           [memSize]uint // Memory
	pc            uint          // Program counter
	ir            uint          // Instruction register
	sr            uint          // Switch register
	lac           uint          // Accumulator register 13th bit is Link flag
	mq            uint          // Multiplier Quotient
	ien           bool          // Whether interrupts are enabled
	pendingIen    bool          // If turning on interrupts is pending
	devices       []device      // Devices for IOT
	deviceNumbers []int         // The device numbers currently registered
}

func New() *PDP8 {
	p := &PDP8{}
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

// Load paper tape in RIM format
func (p *PDP8) LoadRIMTape(tty *TTY, filename string) error {
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

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	// Attach Paper tape in RIM format
	tty.ReaderAttachTape(bufio.NewReader(f))

	// Start of RIM loader
	p.pc = 0o7756

	// Start the punched tape reader
	tty.ReaderStart()

	// Run and panic if HLT
	runNoHlt := func(p *PDP8, cycles int) error {
		hlt, _, err := p.Run(cycles)
		if err != nil {
			return err
		}

		// TODO: This won't work with autostarting RIM tapes
		if hlt {
			panic(fmt.Sprintf("HLT at PC: %04o", p.pc-1))
		}
		return nil
	}

	for !tty.ReaderIsEOF() {
		// Run RIM loader to load the paper tape
		if runNoHlt(p, 100); err != nil {
			return err
		}
	}
	// Stop the punched tape reader
	tty.ReaderStop()

	// Run another time in case finishes between EOF and handling last
	// value read
	if runNoHlt(p, 10000); err != nil {
		return err
	}

	// TODO: This won't work with autostarting RIM tapes
	if !tty.ReaderIsEOF() || !(p.pc == 0o7756 || p.pc == 0o7760) {
		return fmt.Errorf("RIM loader didn't finish, PC: %04o", p.pc)
	}
	return nil
}

// TODO: Remove this and implement a BIN loader
func (p *PDP8) Load(filename string) error {
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

	// TODO: Should we be showing the checksum and other values, unless
	// TODO: debug mode is on?
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

func (p *PDP8) AddDevice(d device) error {
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

// Returns (hlt, cyclesLeft, error)
// TODO: Improve cycle accuracy and return number left/over?
// TODO: Test cyclesLeft
func (p *PDP8) Run(cycles int) (bool, int, error) {
	var err error
	var hlt bool
	var isInterrupt bool

	for cycles > 0 {
		opCode, opAddr := p.fetch()
		hlt, err = p.execute(opCode, opAddr)
		if err != nil || hlt {
			break
		}

		if p.ien {
			for _, d := range p.devices {
				isInterrupt, err = d.interrupt()
				if err != nil {
					break
				}
				if isInterrupt {
					p.mem[0] = p.pc
					p.pc = 1
					p.ien = false
					break
				}
			}
		}

		// The effect of ION is delayed by one instruction
		// TODO: test this
		if p.pendingIen {
			p.ien = true
			p.pendingIen = false
		}

		cycles--
	}
	return hlt, cycles, err
}

// Set Program Counter
func (p *PDP8) SetPC(pc uint) {
	p.pc = mask(pc)
}

// Set Switch Register
func (p *PDP8) SetSR(sr uint) {
	p.sr = mask(sr)
}

// TODO: rename this
func (p *PDP8) Cleanup() {
	fmt.Printf(" PC %04o\r\n", mask(p.pc-1))
}

// fetch returns opCode and opAddr if relevant else 0
func (p *PDP8) fetch() (opCode uint, opAddr uint) {
	p.ir = p.mem[p.pc]
	opCode = (p.ir >> 9) & 0o7
	opAddr = 0

	if opCode <= 5 { // If <= JMP and hence includes an address
		opAddr = p.ir & 0o177
		if (p.ir & 0o200) == 0o200 { // If zero page
			opAddr |= p.pc & 0o7600
		}

		// If indirect
		if (p.ir & 0o400) == 0o400 {
			// If auto increment address
			if (opAddr & 0o7770) == 0o10 {
				p.mem[opAddr] = mask(p.mem[opAddr] + 1)
			}
			opAddr = p.mem[opAddr]
		}
	}

	// TODO: Add a switch to turn this off an on for debugging
	// fmt.Printf("PC: %04o, IR: %04o, opCode, %04o, opAddr: %04o\n", p.pc, p.mem[p.pc], opCode, opAddr)

	p.pc = mask(p.pc + 1)
	return opCode, opAddr
}

// Returns (hltExecuted, error)
func (p *PDP8) execute(opCode uint, opAddr uint) (bool, error) {
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
func (p *PDP8) iot() error {
	var err error
	device := (p.ir >> 3) & 0o77
	iotOp := p.ir & 0o7
	switch device {
	case 0o0: // CPU
		switch iotOp {
		case 0o1: // ION
			p.pendingIen = true
		case 0o2: // IOF
			// IOF is immediate unlike ION
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
func (p *PDP8) opr() bool {
	// TODO: Check order as well as AND/OR combinations
	if (p.ir & 0o400) != 0o400 { // Group 1
		if (p.ir & 0o200) == 0o200 { // CLA
			p.lac &= 0o10000
		}
		if (p.ir & 0o100) == 0o100 { // CLL
			p.lac &= 0o7777
		}
		if (p.ir & 0o40) == 0o40 { // CMA
			p.lac ^= 0o7777
		}
		if (p.ir & 0o20) == 0o20 { // CML
			p.lac ^= 0o10000
		}
		if (p.ir & 0o1) == 0o1 { // IAC
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
	} else if (p.ir & 0o1) != 0o1 { // Group 2
		var sv uint
		// SMA, SPA, SZA, SNA, SNL, SZL
		// TODO: Split this out to make it clearer
		sc := ((p.ir&0o100) == 0o100 && (p.lac&0o4000) == 0o4000) ||
			((p.ir&0o40) == 0o40 && (p.lac&0o7777) == 0) ||
			(p.ir&0o20) == 0o20 && (p.lac&0o10000) == 0o10000
		if sc {
			sv = 0
		} else {
			sv = 0o10
		}
		if sv == (p.ir & 0o10) {
			p.pc = mask(p.pc + 1)
		}
		if (p.ir & 0o200) == 0o200 { // CLA
			p.lac &= 0o10000
		}
		if (p.ir & 0o4) == 0o4 { // OSR
			p.lac |= p.sr
		}
		if (p.ir & 0o2) == 0o2 { // HLT
			return true
		}
	} else { // Group 3
		// TODO: Remove as probably not going to emulate a PDP-8/E?
		// TODO: But then again what about on a PDP-8/I?
		// We store MQ so that MQA and MQL can exchange MQ and AC
		t := p.mq
		if (p.ir & 0o201) == 0o201 { // CLA
			p.lac &= 0o10000
		}
		if (p.ir & 0o21) == 0o21 { // MQL
			p.mq = p.lac & 0o7777
			p.lac &= 0o10000
		}
		if (p.ir & 0o101) == 0o101 { // MQA
			p.lac |= t
		}
	}
	return false
}
