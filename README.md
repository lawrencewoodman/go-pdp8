# gopdp8

A PDP-8 emulator written in Go.

The emulator implements as much as possible only portable instructions used by the family of 8.  Therefore, there are a number of limitations:
  * No Group 3 instructions
  * No instructions to turn on/off individual device interrupts

This keeps the code simpler and means that a program that runs on it is likely to run on any PDP-8, assuming it has enough memory and connected devices.


## Comment Conventions

Throughout the source code the bits are labeled differently to the DEC documentation.  We define bit 0 as the Least Significant Bit.

## Testing

In order to test the emulator you will need to supply digital images of a number of paper tapes.  Please see fixtures/README.md for how to obtain them.

## Documentation

For documentation about the paper tapes used please see docs/README.md for how to obtain them.

## Licence
Copyright (C) 2023, Lawrence Woodman <lwoodman@vlifesystems.com>

This software is licensed under an MIT Licence.  Please see the file, LICENCE.md, for details.
