package fsutil

// A RegionBuilder maps from some high-level structure, such as a list of
// descriptions of files, onto some physical structure, like a filesystem.
type RegionBuilder interface {
	Length() int
	Build(Region)
}
