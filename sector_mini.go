package openmcdf

import (
	"fmt"
	"io"
)

type MiniSector struct {
	id     int
	sector *Sector
	off    int
	next   uint32
	size   int
}

//---------- Sector ----------

func newMiniSector(size int, offset int, s *Sector) *MiniSector {
	return &MiniSector{
		id:     -1,
		size:   size,
		off:    offset,
		sector: s,
		next:   FREESECT,
	}
}

func (this *MiniSector) Close() {
	this.sector = nil
	this.id = -1
	this.off = 0
	this.next = FREESECT
}

func (this *MiniSector) Read(r io.ReaderAt, b []byte) (err error) {
	end := len(b)
	if end > this.size {
		end = this.size
	}
	if err = this.sector.Read(r, this.off, b[:end]); err != nil {
		return
	}
	return
}

func (this *MiniSector) setNext(next uint32) {
	this.next = next
}

func (this *MiniSector) Write(data interface{}) (err error) {
	this.sector.modified = true
	switch v := data.(type) {
	case []uint8:
		l := this.size
		if l > len(v) {
			l = len(v)
		}
		err = this.sector.Write(this.off, v[:l])
	default:
		err = fmt.Errorf("Error write in sector %v data: %v", this.id, v)
	}
	return
}

func (this *MiniSector) String() string {
	comment := ""
	if this.next == FREESECT {
		comment = "free"
	} else if this.next == ENDOFCHAIN {
		comment = "end"
	}
	return fmt.Sprintf("MiniSector {id: %v, sector: %v, next: %v, offset: %v, comment: %s}",
		this.id, this.sector.id, this.next, this.off, comment)
}
