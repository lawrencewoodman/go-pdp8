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
	rawC *rawConsole
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

func (t *tty) iot(ir uint, pc uint, ac uint) (uint, uint, error) {
	var key string
	var err error

	device := (ir >> 3) & 0o77
	switch device {
	case 0o3: // Keyboard
		// NOTE: Operations are executed from right bit to left
		if (ir & 0o1) != 0 { // KSF - Skip if ready
			if t.rawC.isKeyWaiting() {
				pc = mask(pc + 1)
			}
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
			// Put the key in AC without changing L
			ac = (ac & 0o10000) | uint(key[0])
		}
	case 0o4: // Teleprinter
		// NOTE: Operations are executed from right bit to left
		if (ir & 0o1) != 0 { // TSF  - Skip if ready
			// Assume we'll always be ready as we
			// are using a fast display for output
			pc = mask(pc + 1)
		}
		if (ir & 0o2) != 0 { // TCF  - Clear Flag
			// TODO: Implement flags
		}
		if (ir & 0o4) != 0 { // TPC  - Print static
			// Output lower 7 bits of accumulator
			// TODO: Why not 8 bits (0o377) ?
			fmt.Printf("%c", ac&0o177)
		}
	}
	return pc, ac, err
}
