package fsutil

// Region projects a flat, contiguous address space onto arbitrary segments
// of one or more underlying buffers.
//
// The most straightforward Region is a single flat buffer, but a region can
// actually consist of any number of separate buffers that need not be
// contiguous in memory. The buffers, of arbitrary and differing sizes,
// are considered to represent sequential areas of logical memory.
//
// The arbitrary sizing comes at the cost of needing to scan through the
// buffers linearly in order to locate a particular byte. As a consequence,
// it works best to keep the total number of buffers small. When working
// with a very fragmented region, it is a good idea to Slice a smaller part
// to do a number of related operations, since those operations will then
// scan only the relevant sub-slices.
type Region [][]byte

// (maybe later we'll impose a requirement that all buffers except the first
// and last must be the same length, in which case we can avoid the need to
// scan.)

func RegionForBytes(src []byte) Region {
	return Region([][]byte{
		src,
	})
}

func RegionFromString(src string) Region {
	return RegionForBytes([]byte(src))
}

func (r Region) Slice(offset, length int) Region {
	// Search through our buffers to find the one that contains
	// the start offset.
	currentStart := 0
	currentEnd := 0
	var tail Region

	for i, buf := range r {
		currentEnd = currentStart + len(buf)
		if offset >= currentStart && offset < currentEnd {
			// Need to copy tail since we'll potentially modify
			// the first and last element to select exactly the
			// right bytes.
			tail = make(Region, len(r)-i)
			copy(tail, r[i:])
			break
		}
		currentStart += len(buf)
	}

	if tail == nil {
		// Start is after the end of the buffer
		return make([][]byte, 0)
	}

	// Trim off any extra bytes on the front of the tail, so our
	// tail starts at the given offset.
	tail[0] = tail[0][offset-currentStart:]

	// Now we need to find the appropriate place to stop.
	lengthSoFar := 0
	for i, buf := range tail {
		lengthBefore := lengthSoFar
		lengthSoFar += len(buf)
		if lengthSoFar >= length {
			// We've found the last buffer.

			// First, trim off any remaining buffers.
			tail = tail[:i+1]

			// Now trim off any extra bytes on the end of the trailing
			// buffer.
			want := length - lengthBefore
			tail[i] = tail[i][:want]
			break
		}
	}

	return tail
}

func (r Region) Length() int {
	length := 0
	for _, buf := range r {
		length += len(buf)
	}
	return length
}

// Blocks creates a region made of equal-sized blocks that may be
// arbitrarly placed and ordered in the source region. This can represent
// a user file that is stored with its contents split over different parts
// of the disk.
func (r Region) Blocks(size int, mapping []int) Region {
	ret := make([][]byte, 0, len(mapping))

	for _, destBlock := range mapping {
		ret = append(ret, r.Slice(destBlock*size, size)...)
	}

	return ret
}

// Bytes "flattens" a region into a single buffer and returns a slice over that
// buffer.
//
// This creates a copy of all of the data in the region.
func (r Region) Bytes() []byte {
	// We'll take a guess as to what size buffer we need and then
	// adjust as needed.
	// Our assumption is that regions with three or more buffers are
	// probably derived from a Blocks call, and thus all but the first
	// and last regions will be the same length.
	guessLen := 0
	if len(r) > 0 {
		guessLen += len(r[0])
	}
	if len(r) > 1 {
		guessLen += len(r[len(r)-1])
	}
	if len(r) > 2 {
		// Second element is probably the same length as the remaining non-edge
		// elements.
		guessLen += len(r[1])*len(r) - 2
	}

	ret := make([]byte, 0, guessLen)
	for _, buf := range r {
		ret = append(ret, buf...)
	}
	return ret
}

func (r *Region) WriteU8(addr int, val byte) {
	loc := r.Slice(addr, 1)
	// If the caller has request a byte outside of the region
	// then we'll fail here with an out-of-bounds error because
	// the returned slice is empty.
	loc[0][0] = val
}

func (r *Region) ReadU8(addr int) byte {
	loc := r.Slice(addr, 1)
	return loc[0][0]
}

func (r *Region) WriteLE(addr int, bytes int, val uint64) {
	loc := r.Slice(addr, bytes)
	for ofs := 0; ofs < bytes; ofs++ {
		loc.WriteU8(ofs, byte(val))
		val = val >> 8
	}
}

func (r *Region) ReadLE(addr int, bytes int) uint64 {
	loc := r.Slice(addr, bytes)
	val := uint64(0)
	for ofs := bytes - 1; ofs >= 0; ofs-- {
		val = (val << 8) | uint64(loc.ReadU8(ofs))
	}
	return val
}

func (r *Region) WriteBE(addr int, bytes int, val uint64) {
	loc := r.Slice(addr, bytes)
	for ofs := bytes - 1; ofs >= 0; ofs-- {
		loc.WriteU8(ofs, byte(val))
		val = val >> 8
	}
}

func (r *Region) ReadBE(addr int, bytes int) uint64 {
	loc := r.Slice(addr, bytes)
	val := uint64(0)
	for ofs := 0; ofs < bytes; ofs++ {
		val = (val << 8) | uint64(loc.ReadU8(ofs))
	}
	return val
}

func (r *Region) WriteU16LE(addr int, val uint16) {
	r.WriteLE(addr, 2, uint64(val))
}

func (r *Region) ReadU16LE(addr int) uint16 {
	return uint16(r.ReadLE(addr, 2))
}

func (r *Region) WriteU32LE(addr int, val uint32) {
	r.WriteLE(addr, 4, uint64(val))
}

func (r *Region) ReadU32LE(addr int) uint32 {
	return uint32(r.ReadLE(addr, 4))
}

func (r *Region) WriteU64LE(addr int, val uint64) {
	r.WriteLE(addr, 8, val)
}

func (r *Region) ReadU64LE(addr int) uint64 {
	return uint64(r.ReadLE(addr, 8))
}

func (r *Region) WriteU16BE(addr int, val uint16) {
	r.WriteBE(addr, 2, uint64(val))
}

func (r *Region) ReadU16BE(addr int) uint16 {
	return uint16(r.ReadBE(addr, 2))
}

func (r *Region) WriteU32BE(addr int, val uint32) {
	r.WriteBE(addr, 4, uint64(val))
}

func (r *Region) ReadU32BE(addr int) uint32 {
	return uint32(r.ReadBE(addr, 4))
}

func (r *Region) WriteU64BE(addr int, val uint64) {
	r.WriteBE(addr, 8, val)
}

func (r *Region) ReadU64BE(addr int) uint64 {
	return uint64(r.ReadBE(addr, 8))
}

func (r *Region) WriteBytes(start int, src []byte) {
	sub := r.Slice(start, len(src))
	// Unless the caller gave us an array that is too long (which is their
	// error) we should now have a sub-slice whose length matches the length of
	// the buffer.
	ofs := 0
	for _, buf := range sub {
		ofs += copy(buf, src[ofs:])
	}
}

func (r *Region) WriteSubregion(addr int, builder RegionBuilder) {
	length := builder.Length()
	builder.Build(r.Slice(addr, length))
}
