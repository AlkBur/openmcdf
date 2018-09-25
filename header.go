package openmcdf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const HeaderSize = 512

const (
	FREESECT   = uint32(0xFFFFFFFF) //-1
	ENDOFCHAIN = uint32(0xFFFFFFFE) //-2
	FATSECT    = uint32(0xFFFFFFFD) //-3
	DIFSECT    = uint32(0xFFFFFFC)  //-4
)

var (
	OleSignature     = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
	VersionException = errors.New("Unsupported Binary File Format version: Mcdf only supports Compound Files with major version equal to 3 or 4 ")
)

type Header struct {
	signature                    [8]byte  //8 byte
	unused_clsid                 [16]byte //16 byte
	minorVersion                 uint16   //2 byte
	majorVersion                 uint16   //2 byte
	byteOrder                    uint16   //2 byte
	sectorShift                  uint16   //2 byte
	miniSectorShift              uint16   //2 byte
	reserved                     [6]byte  //6 byte
	numDirectorySector           uint32   //4 byte
	numFATSector                 uint32   //4 byte
	firstDirectorySectorLocation uint32   //4 byte
	transactionSignatureNumber   uint32   //4 byte
	miniStreamCutoffSize         uint32   //4 byte
	firstMiniFATSectorLocation   uint32   //4 byte
	numMiniFATSector             uint32   //4 byte
	firstDIFATSectorLocation     uint32   //4 byte
	numDIFATSector               uint32   //4 byte
	headerDIFAT                  [109]uint32
	//---------------------------
	modified bool
}

func newHeader() *Header {
	this := &Header{
		byteOrder:                    0xFFFE, //LittleEndian: Intel byte-ordering
		miniSectorShift:              0x0006, // 64 bytes
		numDirectorySector:           0,
		numFATSector:                 0,
		firstDirectorySectorLocation: ENDOFCHAIN,
		transactionSignatureNumber:   0,
		miniStreamCutoffSize:         0x00001000, //4096 bytes
		firstMiniFATSectorLocation:   ENDOFCHAIN,
		numMiniFATSector:             0,
		firstDIFATSectorLocation:     ENDOFCHAIN,
		numDIFATSector:               0,
	}
	this.setVersion(3)

	copy(this.signature[:], OleSignature)

	for i := 0; i < len(this.headerDIFAT); i++ {
		this.headerDIFAT[i] = FREESECT
	}
	return this
}

func (this *Header) String() string {
	return fmt.Sprintf(`Header{
	signature: %v
	clsid: %v
	minor version: %#x
	major version: %#x
	byte order: %#x
	sector shift: %#x
	mini-sector shift: %#x
	reserved: %v
	number directory sectors: %v
	number FAT sectors: %v
	first directory sector location: %v
	transaction signature number: %v
	mini stream cutoff size: %v
	first mini-FAT sector location: %v
	number mini-FAT sectors: %v
	first DIFAT sector location: %v
	number DIFAT sectors: %v
	difat: [
		%v
	]
}`, this.signature, this.unused_clsid, this.minorVersion, this.majorVersion, this.byteOrder, this.sectorShift,
		this.miniSectorShift, this.reserved, this.numDirectorySector, this.numFATSector,
		this.firstDirectorySectorLocation, this.transactionSignatureNumber, this.miniStreamCutoffSize,
		this.firstMiniFATSectorLocation, this.numMiniFATSector, this.firstDIFATSectorLocation,
		this.numDIFATSector, this.headerDIFAT)
}

func (this *Header) setVersion(ver int) error {
	if ver != 3 && ver != 4 {
		return VersionException
	}
	switch ver {
	case 3:
		this.majorVersion = 0x0003
		this.sectorShift = 0x0009 //512 bytes
		this.minorVersion = 0x003E
	case 4:
		this.majorVersion = 0x0004
		this.sectorShift = 0x000C //4096 bytes
		this.minorVersion = 0x003E
	default:
		return VersionException
	}
	this.modified = true
	return nil
}

func (this *Header) getVersion() int {
	return int(this.majorVersion)
}

func (this *Header) Read(r io.Reader) (err error) {
	defer RecoverError(err)

	//Read
	check(ReadData(r, this.signature[:]))                  //8 byte
	check(ReadData(r, this.unused_clsid[:]))               //16 byte
	check(ReadData(r, &this.minorVersion))                 //2 byte
	check(ReadData(r, &this.majorVersion))                 //2 byte
	check(ReadData(r, &this.byteOrder))                    //2 byte
	check(ReadData(r, &this.sectorShift))                  //2 byte
	check(ReadData(r, &this.miniSectorShift))              //2 byte
	check(ReadData(r, this.reserved[:]))                   //6 byte
	check(ReadData(r, &this.numDirectorySector))           //4 byte
	check(ReadData(r, &this.numFATSector))                 //4 byte
	check(ReadData(r, &this.firstDirectorySectorLocation)) //4 byte
	check(ReadData(r, &this.transactionSignatureNumber))   //4 byte
	check(ReadData(r, &this.miniStreamCutoffSize))         //4 byte
	check(ReadData(r, &this.firstMiniFATSectorLocation))   //4 byte
	check(ReadData(r, &this.numMiniFATSector))             //4 byte
	check(ReadData(r, &this.firstDIFATSectorLocation))     //4 byte
	check(ReadData(r, &this.numDIFATSector))               //4 byte

	idx := 0
	for i := 76; i < HeaderSize; i += 4 {
		check(ReadData(r, &this.headerDIFAT[idx])) //436
		idx++
	}

	check(this.checkSignature())
	check(this.checkVersion())
	check(this.checkSectorSize())
	check(this.checkMiniSectorSize())
	check(this.checkByteOrder())
	check(this.checkUserDefinedFieldSize())

	check(this.chechNumDIFATSector())
	check(this.chechNumMiniSector())

	this.modified = false

	return
}

func (this *Header) checkSignature() error {
	if !bytes.Equal(this.signature[:], OleSignature) {
		return WrongFormat
	}
	return nil
}

func (this *Header) checkVersion() error {
	if this.majorVersion != 3 && this.majorVersion != 4 {
		return VersionException
	}
	return nil
}

func (this *Header) checkSectorSize() error {
	if this.sectorShift != 0x0009 && this.sectorShift != 0x000c {
		return fmt.Errorf("illegal sector size %v", this.sectorShift)
	}
	return nil
}

func (this *Header) checkMiniSectorSize() error {
	if this.miniSectorShift != 0x0006 {
		return fmt.Errorf("illegal mimi sector size %v", this.miniSectorShift)
	}
	return nil
}

func (this *Header) checkByteOrder() error {
	//0xFFFE: indicates Intel byte-ordering
	if this.byteOrder != 0xFFFE {
		return fmt.Errorf("illegal byte order %v", this.byteOrder)
	}
	return nil
}

func (this *Header) checkUserDefinedFieldSize() error {
	if this.miniStreamCutoffSize != 0x00001000 {
		return fmt.Errorf("illegal user-defined data size %v", this.miniStreamCutoffSize)
	}
	return nil
}

func (this *Header) Bytes() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	if err := binary.Write(buf, binary.LittleEndian, this); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (this *Header) sectorSize() int {
	return (1 << this.sectorShift)
}

func (this *Header) miniSectorSize() int {
	return (1 << this.miniSectorShift)
}

func (this *Header) FATEntry() int {
	//If Header Major Version is 3, there MUST be 128 fields specified to fill a 512-byte sector.
	//If Header Major Version is 4, there MUST be 1,024 fields specified to fill a 4,096-byte sector.
	return this.sectorSize() / UInt32Size
}

func (this *Header) DirEntry() int {
	// There are 4 directory entries in a 512-byte directory sector (version 3 compound file),
	// and there are 32 directory entries in a 4,096-byte directory sector (version 4 compound file)
	return this.sectorSize() / DirectorySize
}

func (this *Header) chechNumDIFATSector() error {
	// check for DIFAT overflow
	if int32(this.numDIFATSector) < 0 {
		return fmt.Errorf("DIFAT int overflow %v", int32(this.numDIFATSector))
	}
	sz := (this.sectorSize() / 4) - 1
	if int(this.numDIFATSector)*sz+109 > int(this.numFATSector)+sz {
		return fmt.Errorf("num DIFATs exceeds FAT sectors %v", int32(this.numDIFATSector))
	}
	return nil
}

func (this *Header) chechNumMiniSector() error {
	// check for mini FAT overflow
	if int32(this.numMiniFATSector) < 0 {
		return fmt.Errorf("mini FAT int overflow: %v", int64(this.numMiniFATSector))
	}
	return nil
}
