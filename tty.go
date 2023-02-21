/*
 * A TTY device
 *
 * This will emulate a TTY device connected to the console.
 *
 * TODO: Be more specific about what TTY device
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
	ttoReadyFlag        bool // TTO printer
	interruptWaiting    bool // If an interrupt is waiting to be processed
	reader              io.ReadCloser
}

func NewTTY(r io.ReadCloser) *TTY {
	t := &TTY{reader: r, ttiInputBufferEmpty: true}
	return t
}

func (t *TTY) Close() error {
	return t.reader.Close()
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
		n, err := t.reader.Read(key)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 1 {
			t.ttiInputBuffer = key[0]
			t.ttiInputBufferEmpty = false
			t.interruptWaiting = true
			t.ttiReadyFlag = true
			return nil
		}
	}
	if t.ttoReadyFlag {
		t.interruptWaiting = true
	}
	return nil
}

func (t *TTY) deviceNumbers() []int {
	return []int{0o3, 0o4}
}

func (t *TTY) iot(ir uint, pc uint, lac uint) (uint, uint, error) {
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
			t.poll()
			t.ttiReadyFlag = false
			lac = (lac & 0o10000) // Zero AC but keep L
		}
		if (ir & 0o4) != 0 { // KRS - Read static
			if !t.ttiInputBufferEmpty {
				if t.ttiInputBuffer == 0x1C { // Exit on CTRL-\
					fmt.Println("Quit")
					os.Exit(0)
					// TODO: use a flag to exit nicely
				}

				// TODO: Make this lower 7 bits
				// OR the key with the lower 8 bits of AC without changing L
				// NOTE: Bit 8 (MSB left) is set to 1 for keyboard input
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
			fmt.Printf("%c", lac&0o177)
			t.ttoReadyFlag = true
		}
	}
	return pc, lac, err
}
