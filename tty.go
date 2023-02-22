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
	"fmt"
	"io"
	"os"
)

type TTY struct {
	ttiReadyFlag        bool // TTI keyboard
	ttiInputBuffer      byte // A value in the input buffer
	ttiInputBufferEmpty bool // If the input buffer is empty
	ttiIsReaderInput    bool // True if paper tape reader is being used for input
	ttiIsReaderEOF      bool // True if no more tape to read by reader
	ttoReadyFlag        bool // TTO printer
	interruptWaiting    bool // If an interrupt is waiting to be processed

	curin   io.Reader // The current input source
	conin   io.Reader // Console input
	conout  io.Writer // Console output
	tapein  io.Reader // Paper tape reader input
	tapeout io.Writer // Paper tape punch output
}

func NewTTY(conin io.Reader, conout io.Writer) *TTY {
	return &TTY{ttiInputBufferEmpty: true,
		conin: conin, conout: conout,
		curin: conin}
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

// Return if there is an interrupt raised
func (t *TTY) interrupt() bool {
	// TODO: do something with poll error
	t.poll()
	defer func() { t.interruptWaiting = false }()
	return t.interruptWaiting
}

// Checks for activity on device
func (t *TTY) poll() error {
	key := make([]byte, 1)
	if t.ttiInputBufferEmpty {
		n, err := t.curin.Read(key)
		if err != nil && err != io.EOF {
			return err
		}
		if err == io.EOF {
			t.ttiIsReaderEOF = true
		}
		if n == 1 {
			t.ttiInputBuffer = key[0]
			t.ttiInputBufferEmpty = false
			t.interruptWaiting = true
			t.ttiReadyFlag = true
			return nil
		}
	} else {
		t.ttiReadyFlag = true
	}
	if t.ttoReadyFlag {
		t.interruptWaiting = true
	}
	return nil
}

func (t *TTY) deviceNumbers() []int {
	return []int{0o3, 0o4}
}

// Returns PC, LAC, error
func (t *TTY) iot(ir uint, pc uint, lac uint) (uint, uint, error) {
	var err error

	// Operations are executed from right bit to left
	device := (ir >> 3) & 0o77
	switch device {
	case 0o3: // Keyboard
		if err := t.poll(); err != nil {
			return pc, lac, err
		}
		if (ir & 0o1) != 0 { // KSF - Skip if ready
			if t.ttiReadyFlag {
				pc = mask(pc + 1)
			}
		}
		if (ir & 0o2) != 0 { // KCC - Clear AC and Flag
			t.ttiReadyFlag = false
			lac = (lac & 0o10000) // Zero AC but keep L
		}
		if (ir & 0o4) != 0 { // KRS - Read static
			if !t.ttiInputBufferEmpty {
				// Exit on CTRL-\ from keyboard
				if !t.ttiIsReaderInput && t.ttiInputBuffer == 0x1C {
					fmt.Println("Quit")
					os.Exit(0)
					// TODO: use a flag to exit nicely
				}

				// TODO: Make this lower 7 bits
				// OR the key with the lower 8 bits of AC without changing L
				// NOTE: Bit 8 (LSB bit 0) is set to 1 for keyboard input
				lac = lac | (uint(t.ttiInputBuffer) & 0o377)
				t.ttiInputBufferEmpty = true
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
			// Output lower 7 bits of accumulator
			// TODO: Why not 8 bits (0o377) ?
			n, err := t.conout.Write([]byte{byte(lac & 0o177)})
			if err != nil {
				return pc, lac, fmt.Errorf("conout: %s", err)
			}
			if n != 1 {
				return pc, lac, fmt.Errorf("conout: write failed")
			}
			t.ttoReadyFlag = true
		}
	}
	// TODO: find better way of updating pc and lac
	return pc, lac, err
}
