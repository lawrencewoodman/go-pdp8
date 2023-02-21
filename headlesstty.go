/*
 * A headless TTY device used for testing
 *
 * This will emulate a TTY device connected to the console but is
 * used for headless operation so that it can be used for testing.
 *
 * Copyright (C) 2023 Lawrence Woodman <lwoodman@vlifesystems.com>
 *
 * Licensed under an MIT licence.  Please see LICENCE.md for details.
 */

// TODO:  Implement receiving info from device and sending to it
// TODO: Put in separate package?
package pdp8

import (
	"os"
)

type headlessTty struct {
	ttiReadyFlag     bool   // TTI keyboard
	ttoReadyFlag     bool   // TTO printer
	interruptWaiting bool   // If an interrupt is waiting to be processed
	readerTape       []byte // The contents of a tape attached to the reader
	readerTapePos    int    // The position on the tape being read
}

func newHeadlessTty() (*headlessTty, error) {
	t := &headlessTty{}
	return t, nil
}

func (t *headlessTty) close() {
}

func (t *headlessTty) isEOF() bool {
	return t.readerTapePos >= len(t.readerTape)
}

// Read a file and attach it to the paper tape reader
func (t *headlessTty) attachReaderTape(filename string) error {
	var err error
	t.readerTapePos = 0
	t.readerTape, err = os.ReadFile(filename)
	return err
}

// Return if there is an interrupt raised
func (t *headlessTty) interrupt() bool {
	t.poll()
	defer func() { t.interruptWaiting = false }()
	return t.interruptWaiting
}

// Checks for activity on device
func (t *headlessTty) poll() {
	if !t.ttiReadyFlag {
		if t.readerTapePos < len(t.readerTape) {
			t.interruptWaiting = true
			t.ttiReadyFlag = true
			return
		}
	}

	if t.ttoReadyFlag {
		t.interruptWaiting = true
	}
}

func (t *headlessTty) deviceNumbers() []int {
	return []int{0o3, 0o4}
}

func (t *headlessTty) iot(ir uint, pc uint, lac uint) (uint, uint, error) {
	var err error

	// Operations are executed from right bit to left
	device := (ir >> 3) & 0o77
	switch device {
	case 0o3: // Keyboard
		if (ir & 0o1) != 0 { // KSF - Skip if ready
			t.poll()
			if t.ttiReadyFlag {
				pc = mask(pc + 1)
			}
		}
		if (ir & 0o2) != 0 { // KCC - Clear AC and Flag
			t.ttiReadyFlag = false
			lac = (lac & 0o10000) // Zero AC but keep L
		}
		if (ir & 0o4) != 0 { // KRS - Read static
			if t.readerTapePos < len(t.readerTape) {
				key := t.readerTape[t.readerTapePos]
				t.readerTapePos++
				// OR the key with the lower 8 bits of AC without changing L
				lac = lac | (uint(key) & 0o377)
			}
		}
	case 0o4: // Teleprinter
		if (ir & 0o1) != 0 { // TSF  - Skip if ready
			if t.ttoReadyFlag {
				pc = mask(pc + 1)
			}
		}
		if (ir & 0o2) != 0 { // TCF  - Clear Flag
			t.ttoReadyFlag = false
		}
		if (ir & 0o4) != 0 { // TPC  - Print static
			// TODO: Record this so it can be queried
			t.ttoReadyFlag = true
		}
	}
	return pc, lac, err
}
