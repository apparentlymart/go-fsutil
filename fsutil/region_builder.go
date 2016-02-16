package fsutil

// A RegionBuilder maps from some high-level structure, such as a list of
// descriptions of files, onto some physical structure, like a filesystem.
type RegionBuilder interface {
	Length() int
	Build(Region)
}

// A BufferRegionBuilder builds a region from a fixed memory buffer.
type BufferRegionBuilder struct {
	Buffer []byte
}

func (rb *BufferRegionBuilder) Length() int {
	return len(rb.Buffer)
}

func (rb *BufferRegionBuilder) Build(r Region) {
	r.WriteBytes(0, rb.Buffer)
}
