package vfat

import (
	"fmt"

	"github.com/apparentlymart/go-fsutil/fsutil"
)

var BasicSignature = []byte{
	0xeb, 0x58, 0x90, 'M', 'S', 'W', 'I', 'N', '4', '.', '1',
}

const ExtSignature = uint8(0x29)
const BootableSignature = uint16(0xaa55)

var FSTypeSignature = []byte("FAT32   ")

var FSInfoSignature1 = []byte{0x52, 0x52, 0x61, 0x41}
var FSInfoSignature2 = []byte{0x72, 0x72, 0x41, 0x61}
var FSInfoSignature3 = []byte{0x00, 0x00, 0x55, 0xaa}

const FATID = 0x0ffffff8
const EndOfChain uint32 = 0x0fffffff

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
	Label             [11]byte
	ExtraClusterCount int

	RootDir *Directory
}

type layout struct {
	DataClusters int
	FATSize int
	OverheadSize int
	OverheadClusters int
	TotalClusters int
}

func (fs *Filesystem) calcLayout() *layout {
	dataClusters := fs.RootDir.TotalClusters()
	fatSize := (dataClusters * fatEntrySize)
	if fatSize < (2 * fatEntrySize) {
		// FAT must always be at least two entries long because
		// special values live in the first two FAT entries.
		fatSize = 2 * fatEntrySize
	}
	overhead := fatSize + (reservedSectors * sectorSize)
	overheadClusters := (overhead / clusterSize) + 1
	totalClusters := overheadClusters + fs.RootDir.TotalClusters() + fs.ExtraClusterCount

	return &layout{
		DataClusters: dataClusters,
		FATSize: fatSize,
		OverheadSize: overhead,
		OverheadClusters: overheadClusters,
		TotalClusters: totalClusters,
	}
}

func (fs *Filesystem) Length() int {
	layout := fs.calcLayout()
	return layout.TotalClusters * clusterSize
}

func (fs *Filesystem) Build(region fsutil.Region) {
	bootRecord := region.Slice(0, sectorSize)

	layout := fs.calcLayout()
	sectorsPerFAT := uint32((layout.FATSize / sectorSize) + 1)
	nextCluster := uint32(layout.OverheadClusters)
	totalSectors := uint32(layout.TotalClusters * sectorsPerCluster)
	if nextCluster < 2 {
		// Data can't occupy clusters 0 or 1 because the FAT entries
		// for these clusters are used for other purposes.
		nextCluster = 2
	}
	fmt.Println("Data starts at cluster", nextCluster)
	fmt.Println(sectorsPerFAT, "sectors per FAT")

	// Main Signatures
	bootRecord.WriteBytes(0, BasicSignature)
	bootRecord.WriteU16LE(0x1fe, BootableSignature)

	// BIOS Parameter Block
	bootRecord.WriteU16LE(0x00b, sectorSize)
	bootRecord.WriteU8(0x00d, sectorsPerCluster)
	bootRecord.WriteU16LE(0x00e, reservedSectors)
	bootRecord.WriteU8(0x010, 1)    // Number of FATs
	bootRecord.WriteU16LE(0x011, 0) // Number of root entries not used on FAT32
	bootRecord.WriteU8(0x015, 0xf8) // Media Descriptor (Fixed Disk)
	bootRecord.WriteU16LE(0x018, 1) // Physical sectors per track not used
	bootRecord.WriteU16LE(0x01a, 64) // Number of heads not used
	bootRecord.WriteU32LE(0x020, totalSectors)
	bootRecord.WriteU16LE(0x02a, 0) // Version number
	bootRecord.WriteU32LE(0x02c, nextCluster)
	bootRecord.WriteU32LE(0x024, sectorsPerFAT)
	bootRecord.WriteU16LE(0x030, 1) // Sector of FSInfo
	bootRecord.WriteU8(0x042, ExtSignature)
	bootRecord.WriteU32LE(0x043, fs.VolumeID)
	bootRecord.WriteBytes(0x047, fs.Label[:])
	bootRecord.WriteBytes(0x052, FSTypeSignature)

	fsInfo := region.Slice(sectorSize, sectorSize)
	fsInfo.WriteBytes(0x000, FSInfoSignature1)
	fsInfo.WriteBytes(0x1e4, FSInfoSignature2)
	fsInfo.WriteU32LE(0x1e8, 0xffffffff) // Free data clusters not known yet
	fsInfo.WriteU32LE(0x1ec, 0xffffffff) // No most recent data cluster
	fsInfo.WriteBytes(0x1fc, FSInfoSignature3)

	fat := region.Slice(reservedSectors * sectorSize, layout.FATSize)
	fat.WriteU32LE(0, FATID)
	fat.WriteU32LE(4, EndOfChain) // End of chain marker used elsewhere in FAT
}
