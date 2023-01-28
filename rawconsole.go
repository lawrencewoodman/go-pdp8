/*
 * Accept key presses from the console in raw mode
 *
 * This means we can read single key presses
 *
 * Copyright (C) 2023 Lawrence Woodman <lwoodman@vlifesystems.com>
 *
 * Licensed under an MIT licence.  Please see LICENCE.md for details.
 */

package main

import (
	"bufio"
	"errors"
	"fmt"
	"golang.org/x/term"
	"os"
)

type rawConsole struct {
	state      *term.State // stdin original terminal state
	stdinch    chan string // Channel used to receive key presses
	keyWaiting bool        // If a key is waiting
	key        string      // The last key read
	err        error       // An error if raised
}

func newRawConsole() (*rawConsole, error) {
	r := &rawConsole{}
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

	r.stdinch = make(chan string)
	in := bufio.NewReader(os.Stdin)

	// Create a go routine to check for keys from STDIN
	// and send them to the r.stdinch channel
	go func(ch chan<- string, in *bufio.Reader) {
		var b []byte = make([]byte, 1)
		for {
			_, err := in.Read(b)
			if err != nil {
				r.err = fmt.Errorf("stdin: %s", err)
				break
			}
			ch <- string(b)
		}
	}(r.stdinch, in)

	return r, nil
}

func (r *rawConsole) close() {
	close(r.stdinch)
	if err := term.Restore(int(os.Stdin.Fd()), r.state); err != nil {
		// TODO: return err instead?
		panic(fmt.Sprintf("failed to restore terminal: %s", err))
	}
}

func (r *rawConsole) isKeyWaiting() bool {
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
func (r *rawConsole) getKey() (string, error) {
	var key string
	if r.err != nil {
		return "", r.err
	}
	if r.isKeyWaiting() {
		key = r.key
		r.key = ""
		r.keyWaiting = false
	}
	return key, nil
}
