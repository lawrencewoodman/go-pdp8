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
	ttiInputBuffer      byte // A value in the input buffer
	ttiInterruptWaiting bool // If an interrupt is waiting for TTI to be processed
	ttiIsReaderInput    bool // True if paper tape reader is being used for input
	ttiIsReaderEOF      bool // True if no more tape to read by reader
	ttiIsReaderRun      bool // If reader should run
	ttiReaderPos        int  // The position of the reader on the tape
	ttiReadyFlag        bool // TTI keyboard/reader has read a new value

	// This is used to prevent reads happening too quickly
	ttiPendingReadyFlag bool // If the TTI ready flag is waiting to be turned on

	ttoInterruptWaiting bool // If an interrupt is waiting for TTO to be processed
	ttoIsPunchOutput    bool // True if paper tape punch is being used for output
	ttoReadyFlag        bool // TTO printer is ready for a new value

	curin   io.Reader // The current input source
	curout  io.Writer // The current output destination
	conin   io.Reader // Console input
	conout  io.Writer // Console output
	tapein  io.Reader // Paper tape reader input
	tapeout io.Writer // Paper tape punch output
}

func NewTTY(conin io.Reader, conout io.Writer) *TTY {
	tty := &TTY{conin: conin, conout: conout,
		curin: conin, curout: conout}
	return tty
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
	t.ttoIsPunchOutput = true
	t.curout = t.tapeout
	// TODO: Find out if printer is disabled during punching
	// TODO: It seems obvious that it must be, but check
}

// Tell the paper tape pucnh to stop punching the tape
func (t *TTY) PunchStop() {
	t.ttoIsPunchOutput = false
	t.curout = t.conout
}

// Return if there is an interrupt raised
func (t *TTY) interrupt() (bool, error) {
	err := t.poll()
	return t.ttiInterruptWaiting || t.ttoInterruptWaiting, err
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
		t.ttiPendingReadyFlag = true
		t.ttiIsReaderRun = false
		if t.ttiIsReaderInput {
			t.ttiReaderPos++
		} else {
			// Exit on CTRL-\ from keyboard
			if t.ttiInputBuffer == 0x1C {
				fmt.Println("Quit")
				os.Exit(0)
				// TODO: use a flag to exit nicely
			}
		}
	}
	return nil
}

// Checks for activity on device and run reader if requested
func (t *TTY) poll() error {
	var err error
	if t.ttiPendingReadyFlag {
		t.ttiReadyFlag = true
		t.ttiPendingReadyFlag = false
		t.ttiInterruptWaiting = true
		return nil
	} else {
		if (t.ttiIsReaderInput && t.ttiIsReaderRun) ||
			(!t.ttiIsReaderInput && !t.ttiReadyFlag) {
			err = t.read()
		}
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
		// KCC - Clear AC and Flag and run reader
		kcc := func() {
			t.ttiIsReaderRun = true
			t.ttiReadyFlag = false
			t.ttiInterruptWaiting = false

			// The reader is told to run but it won't have read anything
			// by the time this and any other current microcoded
			// instruction finishes
			lac = (lac & 0o10000) // Zero AC but keep L
		}

		// KRS - Read static
		krs := func() {
			// OR the key with the lower 8 bits of AC without changing L
			lac |= (uint(t.ttiInputBuffer) & 0o377)

			if !t.ttiIsReaderInput {
				// Bit 8 (LSB bit 0) is set to 1 for keyboard input
				// TODO: Check this is correct
				lac |= 0o200
			}
		}

		if (ir & 0o7) == 0o1 { // KSF - Skip if ready
			if t.ttiReadyFlag {
				pc = mask(pc + 1)
			}
		}

		if (ir & 0o7) == 0o2 { // KCC - Clear AC and Flag and run reader
			kcc()
		}

		if (ir & 0o7) == 0o4 { // KRS - Read static
			krs()
		}

		if (ir & 0o7) == 0o6 { // KRB - Read and Begin next read
			kcc()
			krs()
		}
	case 0o4: // Teleprinter
		// TCF  - Clear Flag
		tcf := func() {
			t.ttoReadyFlag = false
			t.ttoInterruptWaiting = false
		}

		// TPC  - Print Character
		tpc := func() error {
			// Use 7 bit mask for keyboard
			ttyMask := uint(0o177)
			if t.ttoIsPunchOutput {
				// Use 8-bit mask for punch
				ttyMask = uint(0o377)
			}

			n, err := t.curout.Write([]byte{byte(lac & ttyMask)})
			if err != nil {
				return fmt.Errorf("TTY: %s", err)
			}
			if n != 1 {
				return errors.New("TTY: write failed")
			}
			// Flag won't become ready until a TPC/TLS has been
			// executed and has output it's value
			t.ttoReadyFlag = true
			t.ttoInterruptWaiting = true
			return nil
		}

		if (ir & 0o7) == 0o1 { // TSF  - Skip if ready
			if t.ttoReadyFlag {
				pc = mask(pc + 1)
			}
		}
		if (ir & 0o7) == 0o2 { // TCF  - Clear Flag
			tcf()
		}
		if (ir & 0o7) == 0o4 { // TPC  - Print Character
			err = tpc()
		}
		if (ir & 0o7) == 0o6 { // TLS  - Load and Start
			tcf()
			err = tpc()
		}
	}
	// TODO: find better way of updating pc and lac
	return pc, lac, err
}
