/*
 * Accept key presses from the console in raw mode
 *
 * This means we can read single key presses without waiting for a
 * newline.  RawTerminalReader implements io.ReadCloser
 *
 * Copyright (C) 2023 Lawrence Woodman <lwoodman@vlifesystems.com>
 *
 * Licensed under an MIT licence.  Please see LICENCE.md for details.
 */

// TODO: Put in separate package?
// TODO: Use via an interface as this isn't relevant for many uses
package pdp8

import (
	"bufio"
	"errors"
	"fmt"
	"golang.org/x/term"
	"os"
)

type RawTerminalReader struct {
	state      *term.State // stdin original terminal state
	stdinch    chan byte   // Channel used to receive key presses
	keyWaiting bool        // If a key is waiting
	key        byte        // The last key read
	err        error       // An error if raised
}

func NewRawTerminalReader() (*RawTerminalReader, error) {
	r := &RawTerminalReader{}
	var err error
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, errors.New("stdin/stdout should be terminal")
	}

	// Ensure we can receive a single keypress without waiting for
	// enter to be pressed
	r.state, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("setting stdin to raw: %s", err)
	}

	r.stdinch = make(chan byte)
	in := bufio.NewReader(os.Stdin)

	// Create a go routine to check for keys from STDIN
	// and send them to the r.stdinch channel
	go func(ch chan<- byte, in *bufio.Reader) {
		var b []byte = make([]byte, 1)
		for {
			_, err := in.Read(b)
			if err != nil {
				r.err = fmt.Errorf("stdin: %s", err)
				break
			}
			ch <- b[0]
		}
	}(r.stdinch, in)

	return r, nil
}

func (r *RawTerminalReader) Close() error {
	close(r.stdinch)
	if err := term.Restore(int(os.Stdin.Fd()), r.state); err != nil {
		return fmt.Errorf("failed to restore terminal: %s", err)
	}
	return nil
}

func (r *RawTerminalReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	for i := 0; i < len(p); i++ {
		key, err := r.getKey()
		if err != nil {
			return 0, err
		}
		if key == 0 {
			return i, nil
		}
		p[i] = key
	}
	return len(p), nil
}

// Returns if a key is waiting to be read
func (r *RawTerminalReader) isKeyWaiting() bool {
	if r.keyWaiting {
		return true
	}
	select {
	case key, ok := <-r.stdinch:
		if ok {
			r.keyWaiting = true
			r.key = key
		}
	default:
		r.keyWaiting = false
	}
	return r.keyWaiting
}

// Returns a string representing the key read
func (r *RawTerminalReader) getKey() (byte, error) {
	var key byte
	if r.err != nil {
		return 0, r.err
	}
	if r.isKeyWaiting() {
		key = r.key
		r.key = 0
		r.keyWaiting = false
	}
	return key, nil
}
