package pdp8

import (
	"bytes"
	"testing"
	"time"
)

// TODO: Using ReadDelay in these tests, need to do tests
// TODO: without it to show it working

func tryIOT(t *testing.T, tty *TTY, ir uint, pc uint, lac uint, wantPC uint, wantLac uint) {
	t.Helper()
	gotPC, gotLac, err := tty.iot(ir, pc, lac)
	if err != nil {
		t.Fatalf("iot: %s", err)
	}
	if gotPC != wantPC {
		t.Errorf("iot - PC got: %05o, want: %05o", gotPC, wantPC)
	}
	if gotLac != wantLac {
		t.Errorf("iot - LAC got: %05o, want: %05o", gotLac, wantLac)
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

	tryIOT(t, tty, KRS, 0, 0, 0, 0)
	tryIOT(t, tty, KRS, 0, 0, 0, 0)

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

	tryIOT(t, tty, KRS, 0, 0, 0, 0)

	// Check that KCC advances tape
	tryIOT(t, tty, KCC, 0, 0, 0, 0)
	time.Sleep(ReadDelay * time.Microsecond)

	tryIOT(t, tty, KRS, 0, 0, 0, 0o73)

	// Check that we read the same value
	tryIOT(t, tty, KRS, 0, 0, 0, 0o73)

	// Check that the value is ored with AC
	tryIOT(t, tty, KRS, 0, 0o300, 0, 0o373)

	// Advance tape
	tryIOT(t, tty, KCC, 0, 0, 0, 0)
	time.Sleep(ReadDelay * time.Microsecond)

	// Check that we read the next value
	tryIOT(t, tty, KRS, 0, 0, 0, 0o10)

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

	tryIOT(t, tty, KRB, 0, 0, 0, 0)
	time.Sleep(ReadDelay * time.Microsecond)

	tryIOT(t, tty, KRB, 0, 0, 0, 0o73)
	time.Sleep(ReadDelay * time.Microsecond)

	tryIOT(t, tty, KRB, 0, 0, 0, 0o10)

	tty.ReaderStop()
}

func TestIOT_tape_KSF_no_skip_if_not_ready(t *testing.T) {
	var KSF uint = 0o6031
	var KCC uint = 0o6032

	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	defer tty.Close()

	paperTape := []byte{0o73, 0o10}
	tty.ReaderAttachTape(bytes.NewReader(paperTape))
	tty.ReaderStop()

	tryIOT(t, tty, KCC, 0, 0, 0, 0)
	tryIOT(t, tty, KSF, 0, 0, 0, 0)
	tryIOT(t, tty, KSF, 0, 0, 0, 0)
	tryIOT(t, tty, KSF, 0, 0, 0, 0)

	tty.ReaderStop()
}

func TestIOT_tape_KSF_skip_if_ready(t *testing.T) {
	var KSF uint = 0o6031
	var KCC uint = 0o6032

	rw := newDummyReadWriter()
	tty := NewTTY(rw, rw)
	defer tty.Close()

	paperTape := []byte{0o73, 0o10}
	tty.ReaderAttachTape(bytes.NewReader(paperTape))
	tty.ReaderStart()

	tryIOT(t, tty, KCC, 0, 0, 0, 0)
	time.Sleep(ReadDelay * time.Microsecond)

	tryIOT(t, tty, KSF, 0, 0, 1, 0)
	tryIOT(t, tty, KSF, 1, 0, 2, 0)
	tryIOT(t, tty, KSF, 4, 0, 5, 0)

	tty.ReaderStop()
}
