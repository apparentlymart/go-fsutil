package vfat

import (
	_ "time"

	"github.com/apparentlymart/go-fsutil/fsutil"
)

var BasicSignature = []byte{
	0xeb, 0x58, 0x90, 'M', 'S', 'W', 'I', 'N', '4', '.', '1',
}

const ExtSignature = uint8(0x28)
const BootableSignature = uint16(0xaa55)

var FSInfoSignature1 = []byte{0x52, 0x52, 0x61, 0x41}
var FSInfoSignature2 = []byte{0x72, 0x72, 0x41, 0x61}
var FSInfoSignature3 = []byte{0x00, 0x00, 0x55, 0xaa}
var FSInfoSignature4 = []byte{0xf8, 0xdd, 0xdd, 0x0f}

const sectorSize = 512
const clusterSize = 4096
const fatEntrySize = 4
const sectorsPerCluster = clusterSize / sectorSize

// Two reserved sectors:
// - Boot record
// - FSInfo
const reservedSectors = 2

type Filesystem struct {
	HiddenSectorCount uint32
	VolumeID          uint32

	RootDir *Directory
}

func (fs *Filesystem) Length() int {
	reservedClusters := ((reservedSectors * sectorSize) / clusterSize) + 1
	dataClusters := fs.RootDir.TotalClusters()
	fatClusters := ((dataClusters * fatEntrySize) / clusterSize) + 1
	totalClusters := reservedClusters + fatClusters + fs.RootDir.TotalClusters()
	return totalClusters * clusterSize
}

func (fs *Filesystem) Build(region fsutil.Region) {
	// Main Signatures
	region.WriteBytes(0, BasicSignature)
	region.WriteU16LE(0x1fe, BootableSignature)

	// BIOS Parameter Block
	region.WriteU16LE(0x00b, sectorSize)
	region.WriteU8(0x00d, sectorsPerCluster)
	region.WriteU16LE(0x00e, reservedSectors)
	region.WriteU8(0x010, 1)    // Number of FATs
	region.WriteU16LE(0x11, 0)  // Number of rootdir entries not used on FAT32
	region.WriteU8(0x015, 0xf8) // Media Descriptor (Fixed Disk)
}
