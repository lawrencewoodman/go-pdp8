/*
 * A TTY device
 *
 * This will emulate a TTY device connected to the console.
 *
 * TODO: Be more specific about what TTY device
 * TODO: Rename to ASR33?
 *
 * Copyright (C) 2023 Lawrence Woodman <lwoodman@vlifesystems.com>
 *
 * Licensed under an MIT licence.  Please see LICENCE.md for details.
 */

// TODO: Put in separate package?
package pdp8

import (
	"errors"
	"fmt"
	"io"
	"os"
)

type TTY struct {
	ttiReadyFlag     bool // TTI keyboard
	ttiInputBuffer   byte // A value in the input buffer
	ttiIsReaderInput bool // True if paper tape reader is being used for input
	ttiIsPunchOutput bool // True if paper tape punch is being used for output
	ttiIsReaderEOF   bool // True if no more tape to read by reader
	ttiIsReaderRun   bool // If reader should run
	ttiReaderPos     int  // The position of the reader on the tape
	ttoReadyFlag     bool // TTO printer
	interruptWaiting bool // If an interrupt is waiting to be processed

	curin   io.Reader // The current input source
	curout  io.Writer // The current output destination
	conin   io.Reader // Console input
	conout  io.Writer // Console output
	tapein  io.Reader // Paper tape reader input
	tapeout io.Writer // Paper tape punch output
}

func NewTTY(conin io.Reader, conout io.Writer) *TTY {
	return &TTY{conin: conin, conout: conout,
		curin: conin, curout: conout}
}

// Closes device but doesn't close any readers/writers
// passed to it
func (t *TTY) Close() error {
	return nil
}

// Attach a punched tape to the reader
func (t *TTY) ReaderAttachTape(tapein io.Reader) {
	t.tapein = tapein
	t.ttiIsReaderEOF = false
	t.ttiReaderPos = 0
}

// Tell the paper tape reader to start reading the tape
func (t *TTY) ReaderStart() {
	t.ttiIsReaderInput = true
	t.curin = t.tapein
	// TODO: Find out if keyboard is disabled during reading
}

// Tell the paper tape reader to stop reading the tape
func (t *TTY) ReaderStop() {
	t.ttiIsReaderInput = false
	t.curin = t.conin
}

// Returns whether the paper tape reader is finishing reading a tape
func (t *TTY) ReaderIsEOF() bool {
	return t.ttiIsReaderEOF
}

// ReaderPos returns the position on the paper tape, starting at 0
func (t *TTY) ReaderPos() int {
	return t.ttiReaderPos
}

// Attach a punched tape to the punch
func (t *TTY) PunchAttachTape(tapeout io.Writer) {
	t.tapeout = tapeout
}

// Tell the paper tape punch to start punching the tape
func (t *TTY) PunchStart() {
	t.ttiIsPunchOutput = true
	t.curout = t.tapeout
	// TODO: Find out if printer is disabled during punching
	// TODO: It seems obvious that it must be, but check
}

// Tell the paper tape pucnh to stop punching the tape
func (t *TTY) PunchStop() {
	t.ttiIsPunchOutput = false
	t.curout = t.conout
}

// Return if there is an interrupt raised
func (t *TTY) interrupt() bool {
	// TODO: do something with poll error
	t.poll()
	defer func() { t.interruptWaiting = false }()
	return t.interruptWaiting
}

// TODO: rename
func (t *TTY) read() error {
	key := make([]byte, 1)
	n, err := t.curin.Read(key)
	if err == io.EOF {
		t.ttiIsReaderEOF = true
	} else if err != nil {
		return err
	}
	if n == 1 {
		t.ttiInputBuffer = key[0]
		t.interruptWaiting = true
		t.ttiReadyFlag = true
		if t.ttiIsReaderInput {
			t.ttiReaderPos++
		}
	}
	return nil
}

// Checks for activity on device and run reader if requested
func (t *TTY) poll() error {
	var err error
	if (t.ttiIsReaderInput && t.ttiIsReaderRun) ||
		(!t.ttiIsReaderInput && !t.ttiReadyFlag) {
		err = t.read()
		t.ttiIsReaderRun = false
	}

	if t.ttoReadyFlag {
		t.interruptWaiting = true
	}
	return err
}

func (t *TTY) deviceNumbers() []int {
	return []int{0o3, 0o4}
}

// Returns PC, LAC, error
func (t *TTY) iot(ir uint, pc uint, lac uint) (uint, uint, error) {
	var err error

	if err := t.poll(); err != nil {
		return pc, lac, err
	}

	// Operations are executed from right bit to left
	device := (ir >> 3) & 0o77
	switch device {
	case 0o3: // Keyboard
		// KSF - Skip if ready
		if (ir & 0o1) == 0o1 {
			if t.ttiReadyFlag {
				pc = mask(pc + 1)
			}
		}

		// KCC - Clear AC and Flag and run reader
		if (ir & 0o2) == 0o2 {
			t.ttiReadyFlag = false
			t.ttiIsReaderRun = true
			// The reader is told to run but it won't have read anything
			// by the time this and any other current microcoded
			// instruction finishes
			lac = (lac & 0o10000) // Zero AC but keep L
		}

		// KRS - Read static
		if (ir & 0o4) == 0o4 {
			// Exit on CTRL-\ from keyboard
			if !t.ttiIsReaderInput && t.ttiInputBuffer == 0x1C {
				fmt.Println("Quit")
				os.Exit(0)
				// TODO: use a flag to exit nicely
			}
			// OR the key with the lower 8 bits of AC without changing L
			lac |= (uint(t.ttiInputBuffer) & 0o377)

			if !t.ttiIsReaderInput {
				// Bit 8 (LSB bit 0) is set to 1 for keyboard input
				// TODO: Check this is correct
				lac |= 0o200
			}
		}
	case 0o4: // Teleprinter
		if (ir & 0o1) == 0o1 { // TSF  - Skip if ready
			if t.ttoReadyFlag {
				pc = mask(pc + 1)
			}
		}
		if (ir & 0o2) == 0o2 { // TCF  - Clear Flag
			t.ttoReadyFlag = false
		}
		if (ir & 0o4) == 0o4 { // TPC  - Print static
			ttyMask := uint(0o177) // Use 7 bit mask for keyboard
			if t.ttiIsPunchOutput {
				// Use 8-bit mask for punch
				ttyMask = uint(0o377)
			}

			n, err := t.curout.Write([]byte{byte(lac & ttyMask)})
			if err != nil {
				return pc, lac, fmt.Errorf("TTY: %s", err)
			}
			if n != 1 {
				return pc, lac, errors.New("TTY: write failed")
			}
			t.ttoReadyFlag = true
		}
	}
	// TODO: find better way of updating pc and lac
	return pc, lac, err
}
