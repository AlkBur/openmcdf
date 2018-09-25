package openmcdf

import (
	"errors"
	"fmt"
)

const (
	MemoryMini TypeMiniMemory = iota
	MemoryMiniFree
	MemmoryMiniCount
)

type TypeMiniMemory uint8

type MiniMemory struct {
	data       [][]*MiniSector
	set        []map[*MiniSector]string
	sectorSize int
}

type MiniMemoryIterator struct {
	current int
	size    int
	data    []*MiniSector
}

func newMiniMemory(sectorSize int) *MiniMemory {
	this := &MiniMemory{
		set:        make([]map[*MiniSector]string, MemmoryMiniCount),
		sectorSize: sectorSize,
		data:       make([][]*MiniSector, MemmoryMiniCount),
	}
	for i := TypeMiniMemory(0); i < MemmoryMiniCount; i++ {
		this.data[i] = make([]*MiniSector, 0, 10)
		this.set[i] = make(map[*MiniSector]string)

	}
	return this
}

func (this *MiniMemory) addSector(s *MiniSector, t TypeMiniMemory) error {
	if s.size != this.sectorSize {
		return fmt.Errorf("error in the size of a mini-sector: %v", s)
	}
	_, ok := this.set[t][s]
	if ok {
		return fmt.Errorf("MiniSector already added in %v: %v", t, s)
	}
	switch t {
	case MemoryMini:
		if s.sector.sectorType != TypeSectorMiniFAT {
			s.sector.sectorType = TypeSectorMiniFAT
		}
		s.id = this.Len(t)
	case MemoryMiniFree:
		_, ok := this.set[MemoryMini][s]
		if !ok {
			return fmt.Errorf("MiniSector not found in mini-memory: %v", t, s)
		}
		s.next = FREESECT
	default:
		return fmt.Errorf("Unknown type mini-memory: %v", t)
	}
	this.data[t] = append(this.data[t], s)
	this.set[t][s] = t.String()
	return nil
}

func (this *MiniMemory) Close() {
	for i := TypeMiniMemory(0); i < MemmoryMiniCount; i++ {
		for j, s := range this.data[i] {
			delete(this.set[i], s)
			this.data[i][j] = nil
		}
		this.data[i] = nil
		this.set[i] = nil
	}
	this.data = nil
	this.set = nil
	//----
	this.sectorSize = 0
}

func (this *MiniMemory) Len(t TypeMiniMemory) int {
	if t >= MemmoryMiniCount {
		return 0
	}
	return len(this.data[t])
}

func (this *MiniMemory) NewIterator(t TypeMiniMemory) *MiniMemoryIterator {
	if t >= MemmoryMiniCount {
		return nil
	}
	return &MiniMemoryIterator{current: -1, data: this.data[t]}
}

func (this TypeMiniMemory) String() string {
	switch this {
	case MemoryMini:
		return "MiniFAT"
	case MemoryMiniFree:
		return "Free MiniFAT"
	}
	return "Unknown"
}

func (this *MiniMemoryIterator) Value() *MiniSector {
	if this.current >= len(this.data) || this.current < 0 {
		return nil
	}
	return this.data[this.current]
}
func (this *MiniMemoryIterator) Next() bool {
	this.current++
	if this.current >= len(this.data) {
		return false
	}
	return true
}

func (this *MiniMemory) getLastSector(t TypeMiniMemory) *MiniSector {
	if this.Len(t) > 0 {
		s, err := this.Get(t, this.Len(t)-1)
		if err == nil {
			return s
		}
	}
	return nil
}

func (this *MiniMemory) Get(t TypeMiniMemory, id int) (*MiniSector, error) {
	if id >= this.Len(t) || id < 0 {
		return nil, fmt.Errorf("Index out of range (%v): %v", t, id)
	}
	return this.data[t][id], nil
}

func (this *MiniMemory) Pop() (*MiniSector, error) {
	if this.Len(MemoryMiniFree) <= 0 {
		return nil, errors.New("Free mini-memory is empty")
	}
	s := this.data[MemoryMiniFree][0]
	this.data[MemoryMiniFree] = this.data[MemoryMiniFree][1:]
	return s, nil
}

func (this *MiniMemory) Push(s *MiniSector) (err error) {
	err = this.addSector(s, MemoryMiniFree)
	return
}
