package fsutil

import (
	_ "reflect"
	"testing"
)

func TestOpen(t *testing.T) {
	rf, err := OpenFile("testdata/test.bin", ReadOnly)
	if err != nil {
		t.Errorf("failed to open file: %s", err)
		return
	}

	val := rf.Region.ReadU16BE(8)
	if val != uint16(0xbead) {
		t.Errorf("got 0x%04x from file; want 0xbead", val)
	}

	err = rf.Close()
	if err != nil {
		t.Errorf("failed to close file: %s", err)
	}
}
