package openmcdf

import (
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

//SectorType
const (
	TypeSectorMemmoryFAT SectorType = iota
	TypeSectorMemmoryMiniFAT
	TypeSectorMemmoryDirectory
	TypeSectorMemmoryDIFAT
	TypeSectorFAT
	TypeSectorMiniFAT
)

type SectorType uint8

type SectorCollection struct {
	data []*Sector
}

type SectorIterator struct {
	current int32
	data    *SectorCollection
}

type Sector struct {
	id         int
	size       int
	data       []byte
	changed    bool
	next       uint32
	sectorType SectorType
	///
	modified bool
}

//---------- Sector collection ----------

func (this SectorType) String() string {
	switch this {
	case TypeSectorMemmoryFAT:
		return "Table FAT"
	case TypeSectorMemmoryMiniFAT:
		return "Table MiniFAT"
	case TypeSectorMemmoryDirectory:
		return "Directory"
	case TypeSectorMiniFAT:
		return "MiniFAT"
	case TypeSectorMemmoryDIFAT:
		return "DIFAT"
	}
	return "FAT"
}

//---------- Sector collection ----------

func newSectorCollection(sectroSize, count int) *SectorCollection {
	this := &SectorCollection{
		data: make([]*Sector, count),
	}
	for i := 0; i < count; i++ {
		s := newSector(sectroSize)
		s.id = i
		this.data[i] = s
	}
	return this
}

func (this *SectorCollection) Close() {
	for i := range this.data {
		if this.data[i] != nil {
			this.data[i].Close()
			this.data[i] = nil
		}
	}
	this.data = nil
}

func (this *SectorCollection) Len() int {
	return len(this.data)
}

func (this *SectorCollection) Add(s *Sector) {
	s.id = this.Len()
	this.data = append(this.data, s)
}

func (this *SectorCollection) Get(SecID int32) (s *Sector, err error) {
	if SecID < 0 || int(SecID) >= this.Len() {
		return nil, fmt.Errorf("Sector collection: Index out of range: %v", SecID)
	}
	return this.data[SecID], nil
}

//func (this *SectorCollection)getBytes(SecID int32) ([]byte, error) {
//	s, err := this.Get(SecID)
//	if err != nil {
//		return nil, err
//	}else if s == nil {
//		return nil, fmt.Errorf("Sector in collection is null: %v", SecID)
//	}
//	return s.Read(this.r)
//}

func (this *SectorCollection) String() string {
	str := strings.Builder{}
	str.WriteString("Sectors: [")
	for _, s := range this.data {
		str.WriteString("\n\t")
		str.WriteString(s.String())
	}
	if len(this.data) > 0 {
		str.WriteString("\n")
	}
	str.WriteString("]")
	return str.String()
}

func (this *SectorCollection) Iterator() *SectorIterator {
	return &SectorIterator{current: -1, data: this}
}

func (this *SectorIterator) Value() *Sector {
	s, err := this.data.Get(this.current)
	if err != nil {
		return nil
	}
	return s
}
func (this *SectorIterator) Next() bool {
	this.current++
	if this.current >= int32(this.data.Len()) {
		return false
	}
	return true
}

//---------- Sector ----------

func newSector(size int) *Sector {
	return &Sector{
		id:         -1,
		size:       size,
		next:       FREESECT,
		modified:   false,
		sectorType: TypeSectorFAT,
	}
}

func (this *Sector) Close() {
	this.data = nil
	this.id = -1
	this.next = FREESECT
	this.sectorType = TypeSectorFAT
}

func (this *Sector) Read(r io.ReaderAt, off int, b []byte) (err error) {
	if this.data == nil {
		var n int
		this.data = make([]byte, this.size)
		off := int64(HeaderSize) + int64(this.id)*int64(this.size)
		n, err = r.ReadAt(this.data, off)
		if err != nil {
			this.data = nil
			return
		} else if n < this.size {
			this.data = nil
			err = fmt.Errorf("Read less than sector size: %v", n)
			return
		}
	}
	if b != nil {
		copy(b, this.data[off:])
	}
	return
}

func (this *Sector) read(r io.ReaderAt) (err error) {
	if this.data == nil {
		var n int
		this.data = make([]byte, this.size)
		off := int64(HeaderSize) + int64(this.id)*int64(this.size)
		n, err = r.ReadAt(this.data, off)
		if err != nil {
			this.data = nil
			return
		} else if n < this.size {
			this.data = nil
			err = fmt.Errorf("Read less than sector size: %v", n)
			return
		}
	}
	return
}

func (this *Sector) setNext(next uint32) {
	this.next = next
}

func (this *Sector) String() string {
	comment := ""
	if this.next == FATSECT {
		comment = "fat"
	} else if this.next == FREESECT {
		comment = "free"
	} else if this.next == ENDOFCHAIN {
		comment = "end"
	}
	if comment == "" {
		return fmt.Sprintf("Sector {id: %v, next: %v, type: %v}",
			this.id, this.next, this.sectorType)
	}
	return fmt.Sprintf("Sector {id: %v, next: %v, type: %v, comment: %s}",
		this.id, this.next, this.sectorType, comment)
}

func (this *Sector) Write(offset int, data interface{}) (err error) {
	if this.data == nil {
		err = fmt.Errorf("Error write in sector: %v", this.id)
		return
	} else if data == nil {
		err = fmt.Errorf("Error write nill in sector: %v", this.id)
		return
	} else if offset >= len(this.data) {
		err = fmt.Errorf("Error write nill in sector: %v, offset: %v", this.id, offset)
		return
	}

	this.modified = true
	switch v := data.(type) {
	case *Directory:
		var buf []byte
		if buf, err = v.Bytes(); err != nil {
			return
		}
		if n := copy(this.data[offset:], buf); n != DirectorySize {
			err = fmt.Errorf("Error write directory: %v", n)
		}
	case uint32:
		binary.LittleEndian.PutUint32(this.data[offset:], uint32(v))
	case []uint8:
		l := len(v)
		if l > this.size {
			l = this.size
		}
		if n := copy(this.data[offset:], v); n != l {
			err = fmt.Errorf("Error write bytes in directory: %v", n)
		}
	default:
		err = fmt.Errorf("Error write in sector %v data: %v", this.id, v)
	}
	return
}
