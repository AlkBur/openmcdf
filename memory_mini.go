package openmcdf

import (
	"fmt"
)

type MiniMemory struct {
	data       []*MiniSector
	free       map[*MiniSector]bool
	sectorSize int
}

type MiniMemoryIterator struct {
	current int
	size    int
	data    []*MiniSector
}

func newMiniMemory(sectorSize int) *MiniMemory {
	this := &MiniMemory{
		free:       make(map[*MiniSector]bool),
		sectorSize: sectorSize,
		data:       make([]*MiniSector, 0, 10),
	}
	return this
}

func (this *MiniMemory) addSector(s *MiniSector) error {
	if s.size != this.sectorSize {
		return fmt.Errorf("error in the size of a mini-sector: %v", s)
	}
	if s.id >= 0 {
		return fmt.Errorf("MiniSector has already been added: %v", s)
	}

	if s.sector.sectorType != TypeSectorMiniFAT {
		s.sector.sectorType = TypeSectorMiniFAT
	}
	s.id = this.Len()
	this.data = append(this.data, s)
	return nil
}

func (this *MiniMemory) check(s *MiniSector) (err error) {
	id := s.id
	if id < 0 || id >= this.Len() || s != this.data[id] {
		err = fmt.Errorf("Error ID MiniSector: %v", s)
	}
	return
}

func (this *MiniMemory) Delete(s *MiniSector) (err error) {
	if err = this.check(s); err != nil {
		return
	}
	//delete element
	delete(this.free, s)
	this.data = append(this.data[:s.id], this.data[s.id+1:]...)
	return
}

func (this *MiniMemory) Close() {
	for i := range this.data {
		this.data[i] = nil
	}
	for s := range this.free {
		delete(this.free, s)
	}
	this.data = nil
	this.free = nil
	//----
	this.sectorSize = 0
}

func (this *MiniMemory) Len() int {
	return len(this.data)
}

func (this *MiniMemory) NewIterator() *MiniMemoryIterator {
	return &MiniMemoryIterator{current: -1, data: this.data}
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

func (this *MiniMemory) getLastSector() *MiniSector {
	if this.Len() > 0 {
		s, err := this.Get(this.Len() - 1)
		if err == nil {
			return s
		}
	}
	return nil
}

func (this *MiniMemory) Get(id int) (*MiniSector, error) {
	if id >= this.Len() || id < 0 {
		return nil, fmt.Errorf("Index MiniSector out of range: %v", id)
	}
	return this.data[id], nil
}

func (this *MiniMemory) Pop() *MiniSector {
	for s := range this.free {
		delete(this.free, s)
		return s
	}
	return nil
}

func (this *MiniMemory) Push(s *MiniSector) (err error) {
	if err = this.check(s); err != nil {
		return
	}
	_, ok := this.free[s]
	if ok {
		err = fmt.Errorf("Sector already added in free: %v", s)
		return
	}
	s.next = FREESECT
	this.free[s] = true
	return
}
