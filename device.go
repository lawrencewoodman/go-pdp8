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

// TODO: Put in separate package?
package pdp8

// TODO: Have some way of registering device numbers so we don't
// TODO: don't have multiple devices with the same number
type device interface {
	// Is an interrupt raised?
	interrupt() bool
	// Returns PC, LAC, error
	iot(ir uint, pc uint, lac uint) (uint, uint, error)
	// Return a slice of device numbers for the device
	deviceNumbers() []int
	// Close the device when finished with
	// TODO: Check if close best name
	close()
}
