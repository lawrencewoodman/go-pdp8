package pdp8

import (
	"bytes"
	"testing"
)

func tryIOT(t *testing.T, tty *TTY, ir uint, pc uint, lac uint, wantLac uint) {
	_, gotLac, err := tty.iot(ir, pc, lac)
	if err != nil {
		t.Errorf("iot: %s", err)
	}
	if gotLac != wantLac {
		t.Errorf("iot - got: %05o, want: %05o", gotLac, wantLac)
	}
}

// Check that requires KCC to load new value from reader
func TestIOT_tape_KRS_no_KCC(t *testing.T) {
	var KRS uint = 0o6034
	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	defer tty.Close()

	paperTape := []byte{0x73, 0x0A}
	tty.ReaderAttachTape(bytes.NewReader(paperTape))
	tty.ReaderStart()

	tryIOT(t, tty, KRS, 0, 0, 0)
	tryIOT(t, tty, KRS, 0, 0, 0)

	tty.ReaderStop()
}

func TestIOT_tape_KRS_after_KCC(t *testing.T) {
	var KCC uint = 0o6032
	var KRS uint = 0o6034

	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	defer tty.Close()

	paperTape := []byte{0o73, 0o10}
	tty.ReaderAttachTape(bytes.NewReader(paperTape))
	tty.ReaderStart()

	tryIOT(t, tty, KRS, 0, 0, 0)

	// Check that KCC advances tape
	tryIOT(t, tty, KCC, 0, 0, 0)

	tryIOT(t, tty, KRS, 0, 0, 0o73)

	// Check that we read the same value
	tryIOT(t, tty, KRS, 0, 0, 0o73)

	// Check that the value is ored with AC
	tryIOT(t, tty, KRS, 0, 0o300, 0o373)

	// Advance tape
	tryIOT(t, tty, KCC, 0, 0, 0)

	// Check that we read the next value
	tryIOT(t, tty, KRS, 0, 0, 0o10)

	tty.ReaderStop()
}

func TestIOT_tape_KRB(t *testing.T) {
	var KRB uint = 0o6036

	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	defer tty.Close()

	paperTape := []byte{0o73, 0o10}
	tty.ReaderAttachTape(bytes.NewReader(paperTape))
	tty.ReaderStart()

	tryIOT(t, tty, KRB, 0, 0, 0)
	tryIOT(t, tty, KRB, 0, 0, 0o73)
	tryIOT(t, tty, KRB, 0, 0, 0o10)

	tty.ReaderStop()
}
