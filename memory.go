package openmcdf

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	MemoryTableFat TypeMemory = iota
	MemoryTableMini
	MemoryDir
	MemoryDIFAT
	MemoryFree
	MemmoryCount
)

type TypeMemory uint8

type MemoryUInt32Iterator struct {
	current int
	off     int
	size    int
	data    []*Sector
}

type Memory struct {
	data       [][]*Sector
	set        map[*Sector]string
	sectorSize int
}

func newMemory(sectorSize int) *Memory {
	this := &Memory{
		set:        make(map[*Sector]string),
		sectorSize: sectorSize,
		data:       make([][]*Sector, MemmoryCount),
	}
	for i := TypeMemory(0); i < MemmoryCount; i++ {
		this.data[i] = make([]*Sector, 0, 10)
	}
	return this
}

func (this *Memory) addSector(s *Sector, t TypeMemory) error {
	if len(s.data) != this.sectorSize && t != MemoryFree {
		return fmt.Errorf("The sector is not read: %v", s)
	}
	ts, ok := this.set[s]
	if ok {
		return fmt.Errorf("Sector already added in %v: %v", ts, s)
	}
	switch t {
	case MemoryTableFat:
		s.sectorType = TypeSectorMemmoryFAT
		s.next = FATSECT
	case MemoryTableMini:
		s.sectorType = TypeSectorMemmoryMiniFAT
		s.next = ENDOFCHAIN
	case MemoryDir:
		s.sectorType = TypeSectorMemmoryDirectory
		s.next = ENDOFCHAIN
	case MemoryDIFAT:
		s.sectorType = TypeSectorMemmoryDIFAT
		s.next = DIFSECT
	case MemoryFree:
		s.sectorType = TypeSectorFAT
		s.next = FREESECT
	default:
		return fmt.Errorf("Unknown type memory: %v", t)
	}
	this.data[t] = append(this.data[t], s)
	this.set[s] = t.String()
	return nil
}

func (this *Memory) getSector(t TypeMemory, idx int32) (s *Sector, err error) {
	if idx < 0 || int(idx) >= this.Len(t) {
		err = fmt.Errorf("Error get %v memory: %v; len: %v", t, idx, this.Len(t))
	} else {
		s = this.data[t][idx]
	}
	return
}

func (this *Memory) getLastSector(t TypeMemory) (s *Sector) {
	idx := this.Len(t)
	if idx > 0 {
		idx--
		s, _ = this.getSector(t, int32(idx))
	}
	return
}

func (this *Memory) FindSector(s *Sector) bool {
	t, ok := this.set[s]
	if ok {
		Log(t)
	}
	return ok
}

func (this *Memory) Len(t TypeMemory) int {
	if t >= MemmoryCount {
		return 0
	}
	return len(this.data[t])
}

func (this *Memory) CountUint32(t TypeMemory) int {
	if t >= MemmoryCount {
		return 0
	}
	return len(this.data[t]) * this.sectorSize / UInt32Size
}

func (this *Memory) NewUInt32Iterator(t TypeMemory) *MemoryUInt32Iterator {
	if t >= MemmoryCount {
		return nil
	}
	return &MemoryUInt32Iterator{off: -UInt32Size, size: this.sectorSize, data: this.data[t]}
}

func (this *Memory) Close() {
	for i := TypeMemory(0); i < MemmoryCount; i++ {
		for j, s := range this.data[i] {
			delete(this.set, s)
			this.data[i][j] = nil
		}
		this.data[i] = nil
	}
	this.data = nil
	//----
	this.sectorSize = 0
}

func (this *MemoryUInt32Iterator) Value() uint32 {
	s := this.data[this.current]
	return binary.LittleEndian.Uint32(s.data[this.off : this.off+UInt32Size])
}
func (this *MemoryUInt32Iterator) Next() bool {
	this.off += UInt32Size
	if this.off >= this.size {
		this.off = 0
		this.current++
	}
	if this.current >= len(this.data) {
		return false
	}
	return true
}

func (this TypeMemory) String() string {
	switch this {
	case MemoryTableFat:
		return "Table FAT"
	case MemoryTableMini:
		return "Table MiniFAT"
	case MemoryDir:
		return "Directory"
	case MemoryDIFAT:
		return "DIFAT"
	case MemoryFree:
		return "Free FAT"
	}
	return "Unknown"
}

func (this *Memory) changeFAT(s *Sector) (err error) {
	id := s.id
	val := s.next

	if int32(id) < 0 {
		err = fmt.Errorf("Error change table FAT: %v", s)
		return
	}
	if int32(val) < -4 {
		err = fmt.Errorf("Error change table FAT: %v", s)
		return
	}

	sz := this.sectorSize / UInt32Size
	index := int(id) / sz
	offset := int(id) % sz

	var alloc *Sector
	if alloc, err = this.getSector(MemoryTableFat, int32(index)); err != nil {
		err = fmt.Errorf("Error change table FAT: %v", s)
		return
	}
	return alloc.Write(offset*UInt32Size, val)
}

func (this *Memory) changeMiniFAT(mini *MiniSector) (err error) {
	id := mini.id
	val := mini.next

	if int32(id) < 0 {
		err = fmt.Errorf("Error change table mini FAT: %v", mini)
		return
	}
	if int32(val) < -4 {
		err = fmt.Errorf("Error change table mini FAT: %v", mini)
		return
	}

	sz := this.sectorSize / UInt32Size
	index := int(id) / sz
	offset := int(id) % sz

	var alloc *Sector
	if alloc, err = this.getSector(MemoryTableMini, int32(index)); err != nil {
		err = fmt.Errorf("Error change table mini FAT: %v", mini)
		return
	}
	return alloc.Write(offset*UInt32Size, val)
}

func (this *Memory) Pop() (*Sector, error) {
	if this.Len(MemoryFree) <= 0 {
		return nil, errors.New("Free memory is empty")
	}
	s := this.data[MemoryFree][0]
	this.data[MemoryFree] = this.data[MemoryFree][1:]
	delete(this.set, s)
	return s, nil
}

func (this *Memory) Push(s *Sector) (err error) {
	err = this.addSector(s, MemoryFree)
	if s.data != nil {
		s.data = nil
	}
	return
}
