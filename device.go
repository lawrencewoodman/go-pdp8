/*
 * A device interface
 *
 * This will handle devices to be accessed via IOT instruction
 * and raise interrupts
 *
 * Copyright (C) 2023 Lawrence Woodman <lwoodman@vlifesystems.com>
 *
 * Licensed under an MIT licence.  Please see LICENCE.md for details.
 */

package main

// TODO: Have some way of registering device numbers so we don't
// TODO: don't have multiple devices with the same number
type device interface {
	// Returns PC, AC
	iot(ir uint, pc uint, ac uint) (uint, uint, error)
	// Close the device when finished with
	// TODO: Check if close best name
	close()
}
