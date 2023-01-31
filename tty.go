/*
 * A TTY device
 *
 * This will emulate a TTY device connected to the console.
 *
 * Copyright (C) 2023 Lawrence Woodman <lwoodman@vlifesystems.com>
 *
 * Licensed under an MIT licence.  Please see LICENCE.md for details.
 */

package main

import (
	"fmt"
	"os"
)

type tty struct {
	ttiReadyFlag     bool // TTI keyboard
	ttoReadyFlag     bool // TTO printer
	interruptWaiting bool // If an interrupt is waiting to be processed
	rawC             *rawConsole
}

func newTty() (*tty, error) {
	var err error
	t := &tty{}
	t.rawC, err = newRawConsole()
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (t *tty) close() {
	t.rawC.close()
}

// Return if there is an interrupt raised
func (t *tty) interrupt() bool {
	t.poll()
	defer func() { t.interruptWaiting = false }()
	return t.interruptWaiting
}

// Checks for activity on device
func (t *tty) poll() {
	if !t.ttiReadyFlag {
		if t.rawC.isKeyWaiting() {
			t.interruptWaiting = true
			t.ttiReadyFlag = true
			return
		}
	}
	if t.ttoReadyFlag {
		t.interruptWaiting = true
	}
}

func (t *tty) deviceNumbers() []int {
	return []int{0o3, 0o4}
}

func (t *tty) iot(ir uint, pc uint, ac uint) (uint, uint, error) {
	var key string
	var err error

	// NOTE: Operations are executed from right bit to left
	device := (ir >> 3) & 0o77
	switch device {
	case 0o3: // Keyboard
		if (ir & 0o1) != 0 { // KSF - Skip if ready
			t.poll()
			if t.ttiReadyFlag {
				pc = mask(pc + 1)
			}
		}
		if (ir & 0o2) != 0 { // KCC - Clear Flag
			t.ttiReadyFlag = false
			ac = (ac & 0o10000) // Zero AC but keep L
		}
		if (ir & 0o4) != 0 { // KRS - Read static
			key, err = t.rawC.getKey()
			if err != nil {
				return pc, ac, err
			}
			if key[0] == 0x1C { // Exit on CTRL-\
				fmt.Println("Quit")
				os.Exit(0)
				// TODO: use a flag to exit nicely
			}
			// OR the key with the lower 8 bits of AC without changing L
			ac = (ac & 0o10377) | uint(key[0])
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
			fmt.Printf("%c", ac&0o177)
			t.ttoReadyFlag = true
		}
	}
	return pc, ac, err
}
