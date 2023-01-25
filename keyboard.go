/*
 * Keyboard routines
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

type keyboard struct {
	state      *term.State // stdin original terminal state
	stdinch    chan string // Channel used to receive keyboard chars
	keyWaiting bool        // If a key press is waiting
	key        string      // The last key pressed
	err        error       // An error if raised
}

func newKeyboard() (*keyboard, error) {
	k := &keyboard{}
	var err error
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, errors.New("stdin/stdout should be terminal")
	}

	// Ensure we can receive a single keypress without waiting for
	// enter to be pressed
	k.state, err = term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, fmt.Errorf("setting stdin to raw: %s", err)
	}

	k.stdinch = make(chan string)
	in := bufio.NewReader(os.Stdin)

	// Create a go routine to check for keypresses
	// and send them to the k.stdinch channel
	go func(ch chan<- string, in *bufio.Reader) {
		var b []byte = make([]byte, 1)
		for {
			_, err := in.Read(b)
			if err != nil {
				k.err = fmt.Errorf("stdin: %s", err)
				break
			}
			ch <- string(b)
		}
	}(k.stdinch, in)

	return k, nil
}

func (k *keyboard) close() {
	close(k.stdinch)
	if err := term.Restore(int(os.Stdin.Fd()), k.state); err != nil {
		panic(fmt.Sprintf("failed to restore terminal: %s", err))
	}
}

func (k *keyboard) isKeyWaiting() bool {
	if k.keyWaiting {
		return true
	}
	select {
	case ch, ok := <-k.stdinch:
		if ok {
			k.keyWaiting = true
			k.key = ch
		}
	default:
		k.keyWaiting = false
	}
	return k.keyWaiting
}

// Returns a string representing the key pressed
// TODO: Is string appropriate?
func (k *keyboard) getKey() (string, error) {
	var key string
	if k.err != nil {
		return "", k.err
	}
	if k.isKeyWaiting() {
		key = k.key
		k.key = ""
		k.keyWaiting = false
	}
	return key, nil
}
