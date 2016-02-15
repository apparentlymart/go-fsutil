package fsutil

import (
	"reflect"
	"testing"
)

func TestLength(t *testing.T) {
	type lengthTest struct {
		r        Region
		expected int
	}

	tests := []lengthTest{
		{
			Region([][]byte{
				[]byte("Hello"),
			}),
			5,
		},
		{
			Region([][]byte{
				[]byte(""),
			}),
			0,
		},
		{
			Region([][]byte{
				[]byte(""),
				[]byte(""),
			}),
			0,
		},
		{
			Region([][]byte{
				[]byte("Hel"),
				[]byte("lo"),
			}),
			5,
		},
		{
			Region([][]byte{
				[]byte(""),
				[]byte("Hello"),
			}),
			5,
		},
		{
			Region([][]byte{
				[]byte("Hello"),
				[]byte(""),
			}),
			5,
		},
	}

	for _, test := range tests {
		got := test.r.Length()
		if got != test.expected {
			t.Errorf("Length of %#v is %d; want %d", test.r, got, test.expected)
		}
	}
}

func TestBytes(t *testing.T) {
	type test struct {
		r        Region
		expected []byte
	}

	tests := []test{
		{
			Region([][]byte{
				[]byte("Hello"),
			}),
			[]byte("Hello"),
		},
		{
			Region([][]byte{
				[]byte(""),
			}),
			[]byte(""),
		},
		{
			Region([][]byte{
				[]byte(""),
				[]byte(""),
			}),
			[]byte(""),
		},
		{
			Region([][]byte{
				[]byte("Hello"),
				[]byte("World"),
			}),
			[]byte("HelloWorld"),
		},
		{
			Region([][]byte{
				[]byte("Hello"),
				[]byte("Pizza"),
				[]byte("World"),
			}),
			[]byte("HelloPizzaWorld"),
		},
	}

	for _, test := range tests {
		got := test.r.Bytes()
		if !reflect.DeepEqual(got, test.expected) {
			t.Errorf(
				"Bytes in %#v are %#v; want %#v",
				test.r, got, test.expected,
			)
		}
	}
}

func TestSlice(t *testing.T) {
	type sliceTest struct {
		r        Region
		offset   int
		length   int
		expected Region
	}

	tests := []sliceTest{
		{
			Region([][]byte{
				[]byte("Hello"),
			}),
			0,
			0,
			Region([][]byte{
				[]byte{},
			}),
		},
		{
			Region([][]byte{
				[]byte("Hello"),
			}),
			0,
			3,
			Region([][]byte{
				[]byte("Hel"),
			}),
		},
		{
			Region([][]byte{
				[]byte("Hello"),
			}),
			0,
			5,
			Region([][]byte{
				[]byte("Hello"),
			}),
		},
		{
			Region([][]byte{
				[]byte("Hello"),
			}),
			0,
			6,
			Region([][]byte{
				[]byte("Hello"),
			}),
		},
		{
			Region([][]byte{
				[]byte("Hello"),
			}),
			1,
			2,
			Region([][]byte{
				[]byte("el"),
			}),
		},
		{
			Region([][]byte{
				[]byte("Hello"),
			}),
			1,
			5,
			Region([][]byte{
				[]byte("ello"),
			}),
		},
		{
			Region([][]byte{
				[]byte("Hel"),
				[]byte("lo"),
			}),
			0,
			5,
			Region([][]byte{
				[]byte("Hel"),
				[]byte("lo"),
			}),
		},
		{
			Region([][]byte{
				[]byte("Hel"),
				[]byte("lo"),
			}),
			1,
			5,
			Region([][]byte{
				[]byte("el"),
				[]byte("lo"),
			}),
		},
		{
			Region([][]byte{
				[]byte("Hel"),
				[]byte("lo"),
			}),
			3,
			5,
			Region([][]byte{
				[]byte("lo"),
			}),
		},
		{
			Region([][]byte{
				[]byte("Hel"),
				[]byte("lo "),
				[]byte("Wor"),
				[]byte("ld"),
			}),
			0,
			5,
			Region([][]byte{
				[]byte("Hel"),
				[]byte("lo"),
			}),
		},
		{
			Region([][]byte{
				[]byte("Hel"),
				[]byte("lo "),
				[]byte("Wor"),
				[]byte("ld"),
			}),
			1,
			6,
			Region([][]byte{
				[]byte("el"),
				[]byte("lo "),
				[]byte("W"),
			}),
		},
		{
			Region([][]byte{
				[]byte("Hel"),
				[]byte("lo "),
				[]byte("Wor"),
				[]byte("ld"),
			}),
			6,
			6,
			Region([][]byte{
				[]byte("Wor"),
				[]byte("ld"),
			}),
		},
	}

	for _, test := range tests {
		got := test.r.Slice(test.offset, test.length)
		if !reflect.DeepEqual(got, test.expected) {
			t.Errorf(
				"Slice %#v from %d for %d is %#v; want %#v",
				test.r, test.offset, test.length, got, test.expected,
			)
		}
	}
}

func TestBlocks(t *testing.T) {
	type test struct {
		r        Region
		size     int
		mapping  []int
		expected Region
	}

	tests := []test{
		{
			Region([][]byte{
				[]byte("Hello!!!"),
			}),
			4,
			[]int{0, 1},
			Region([][]byte{
				[]byte("Hell"),
				[]byte("o!!!"),
			}),
		},
		{
			Region([][]byte{
				[]byte("Hello!!!"),
			}),
			4,
			[]int{1, 0},
			Region([][]byte{
				[]byte("o!!!"),
				[]byte("Hell"),
			}),
		},
		{
			Region([][]byte{
				[]byte("o!!!FoooHell"),
			}),
			4,
			[]int{2, 0},
			Region([][]byte{
				[]byte("Hell"),
				[]byte("o!!!"),
			}),
		},
	}

	for _, test := range tests {
		got := test.r.Blocks(test.size, test.mapping)
		if !reflect.DeepEqual(got, test.expected) {
			t.Errorf(
				"%#v blocks %d %#v is %#v; want %#v",
				test.r, test.size, test.mapping, got, test.expected,
			)
		}
	}
}

func TestWriteBytes(t *testing.T) {
	reg := Region([][]byte{
		[]byte("...."),
		[]byte("...."),
		[]byte("...."),
		[]byte("...."),
	})

	reg.WriteBytes(2, []byte("12345678"))

	expect := Region([][]byte{
		[]byte("..12"),
		[]byte("3456"),
		[]byte("78.."),
		[]byte("...."),
	})

	if !reflect.DeepEqual(reg, expect) {
		t.Errorf(
			"Result is %#v; want %#v",
			reg, expect,
		)
	}
}

func TestWriteU32LE(t *testing.T) {
	reg := Region([][]byte{
		[]byte{0x00, 0x00, 0x00, 0x00},
		[]byte{0x00, 0x00, 0x00, 0x00},
		[]byte{0x00, 0x00, 0x00, 0x00},
	})

	reg.WriteU32LE(2, 0xdeadbeef)

	expect := Region([][]byte{
		[]byte{0x00, 0x00, 0xef, 0xbe},
		[]byte{0xad, 0xde, 0x00, 0x00},
		[]byte{0x00, 0x00, 0x00, 0x00},
	})

	if !reflect.DeepEqual(reg, expect) {
		t.Errorf(
			"Result is %#v; want %#v",
			reg, expect,
		)
	}
}

func TestWriteU32BE(t *testing.T) {
	reg := Region([][]byte{
		[]byte{0x00, 0x00, 0x00, 0x00},
		[]byte{0x00, 0x00, 0x00, 0x00},
		[]byte{0x00, 0x00, 0x00, 0x00},
	})

	reg.WriteU32BE(2, 0xdeadbeef)

	expect := Region([][]byte{
		[]byte{0x00, 0x00, 0xde, 0xad},
		[]byte{0xbe, 0xef, 0x00, 0x00},
		[]byte{0x00, 0x00, 0x00, 0x00},
	})

	if !reflect.DeepEqual(reg, expect) {
		t.Errorf(
			"Result is %#v; want %#v",
			reg, expect,
		)
	}
}

func TestReadU32LE(t *testing.T) {
	reg := Region([][]byte{
		[]byte{0x00, 0x00, 0xef, 0xbe},
		[]byte{0xad, 0xde, 0x00, 0x00},
	})

	got := reg.ReadU32LE(2)
	expect := uint32(0xdeadbeef)

	if got != expect {
		t.Errorf(
			"Result is %04x; want %04x",
			got, expect,
		)
	}
}

func TestReadU32BE(t *testing.T) {
	reg := Region([][]byte{
		[]byte{0x00, 0x00, 0xde, 0xad},
		[]byte{0xbe, 0xef, 0x00, 0x00},
	})

	got := reg.ReadU32BE(2)
	expect := uint32(0xdeadbeef)

	if got != expect {
		t.Errorf(
			"Result is %04x; want %04x",
			got, expect,
		)
	}
}
