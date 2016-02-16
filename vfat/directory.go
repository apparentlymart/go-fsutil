package vfat

import (
	"time"
	"unicode/utf8"

	"github.com/apparentlymart/go-fsutil/fsutil"
)

const DirEntrySize = 32

type Attributes uint8

const (
	ReadOnlyAttr  Attributes = 0x01
	HiddenAttr    Attributes = 0x02
	SystemAttr    Attributes = 0x04
	VolumeIDAttr  Attributes = 0x08
	DirectoryAttr Attributes = 0x10
	ArchiveAttr   Attributes = 0x20
	LFNAttrs      Attributes = ReadOnlyAttr | HiddenAttr | SystemAttr | VolumeIDAttr
)

type DirEntryCommon struct {
	Name             string
	Attributes       Attributes
	CreationTime     time.Time
	LastAccessedTime time.Time
	LastModifiedTime time.Time
}

type DirEntryDir struct {
	DirEntryCommon

	Directory *Directory
}

type DirEntryFile struct {
	DirEntryCommon

	BodyBuilder fsutil.RegionBuilder
}

type Directory struct {
	Dirs  []DirEntryDir
	Files []DirEntryFile
}

// LFNEntryCount returns the number of additional directory entries
// that are needed to represent this entry's "long filename".
func (e *DirEntryCommon) LFNEntryCount() int {
	// Each LFN entry can have 13 UCS-16 characters, and we need to
	// leave room for the 2-byte null terminator.
	chars := utf8.RuneCountInString(e.Name) + 1
	return chars / 13
}

// TotalSize returns the total size of the directory and all of the
// subdirectories and files it returns to, in clusters.
//
// It takes into account cluster, meaning that all file sizes are rounded
// up to the nearest cluster.
func (d *Directory) TotalClusters(isRoot bool) int {

	// Each directory entry takes 32 bytes
	tableBytes := 0
	dataClusters := 0
	for _, entry := range d.Dirs {
		tableBytes += 32 * (entry.LFNEntryCount() + 1)
		dataClusters += entry.Directory.TotalClusters(false)
	}
	for _, entry := range d.Files {
		tableBytes += 32 * (entry.LFNEntryCount() + 1)
		fileSize := entry.BodyBuilder.Length()
		dataClusters += (fileSize / clusterSize) + 1
	}
	if isRoot {
		// Root directory also contains the volume label record
		tableBytes += 32
	}
	return dataClusters + (tableBytes / clusterSize) + 1
}

func (d *Directory) TableBytes(isRoot bool) int {
	// Each directory entry takes 32 bytes
	tableBytes := 0
	for _, entry := range d.Dirs {
		tableBytes += 32 * (entry.LFNEntryCount() + 1)
	}
	for _, entry := range d.Files {
		tableBytes += 32 * (entry.LFNEntryCount() + 1)
	}
	if isRoot {
		// Root directory also contains the volume label record
		tableBytes += 32
	}
	return tableBytes
}
