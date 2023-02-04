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
	"os"
	"strconv"
)

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
