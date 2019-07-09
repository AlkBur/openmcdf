package openmcdf

import (
	"errors"
	"fmt"
	"os"
)

const UInt32Size = 4

var (
	WrongFormat = errors.New("Wrong file format")
)

type CompoundFile struct {
	f      *os.File
	header *Header
	//memmory
	memory *Memory
	mini   *MiniMemory
	//sectors
	sectors *SectorCollection
	//RBTree
	root      *Storage
	directory *DirectoryCollection
	//other
	miniSectorSize int
	sectorSize     int
}

func New(ver int) (this *CompoundFile, err error) {
	this = &CompoundFile{
		header: newHeader(),
	}
	if err = this.header.setVersion(ver); err != nil {
		this = nil
		return
	}
	this.sectorSize = this.header.sectorSize()
	this.miniSectorSize = this.header.miniSectorSize()

	//Alloc
	this.memory = newMemory(this.sectorSize)
	this.mini = newMiniMemory(this.miniSectorSize)
	this.sectors = newSectorCollection(this.SectorSize(), 0)
	this.directory = newDirectoryCollection(1)

	//Create ROOR ENTRY
	var de *Directory
	de, err = this.directory.New(this, "Root Entry", StgRoot)
	if err = this.updateDirectory(de); err != nil {
		this = nil
		return
	}

	this.root = de.newRootStorage(this)
	return
}

func Open(filename string) (this *CompoundFile, err error) {
	var fileInfo os.FileInfo
	fileInfo, err = os.Stat(filename)
	if err != nil {
		return
	} else {
		if fileInfo.Size() < 512 {
			err = WrongFormat
			return
		}
		var f *os.File
		if f, err = os.OpenFile(filename, os.O_RDWR, 0644); err != nil {
			return
		}
		this = &CompoundFile{
			header: &Header{},
			f:      f,
		}
		if err = this.header.Read(f); err != nil {
			return
		}
		this.sectorSize = this.header.sectorSize()
		this.miniSectorSize = this.header.miniSectorSize()

		this.memory = newMemory(this.sectorSize)
		this.mini = newMiniMemory(this.miniSectorSize)

		n := int((fileInfo.Size() - HeaderSize) / int64(this.SectorSize()))
		size := this.SectorSize()
		this.sectors = newSectorCollection(size, n)
	}
	err = this.load()
	return
}

func (this *CompoundFile) Header() *Header {
	return this.header
}

func (this *CompoundFile) SectorSize() int {
	return this.sectorSize
}

func (this *CompoundFile) MiniSectorSize() int {
	return this.miniSectorSize
}

func (this *CompoundFile) load() (err error) {
	//FAT
	if err = this.readFAT(); err != nil {
		return
	}

	//Directory
	if err = this.readDirectory(); err != nil {
		return
	}
	//MiniFAT start in root directory
	if err = this.readMiniFAT(); err != nil {
		return
	}

	var de *Directory
	if de, err = this.directory.Get(0); err != nil {
		err = fmt.Errorf("Error get root directory: %v", err)
		return
	}
	this.root = de.newRootStorage(this)

	//Log(this.directory)

	return
}

func (this *CompoundFile) readFAT() (err error) {
	var s *Sector

	for i := 0; i < int(this.header.numFATSector); i++ {
		if s, err = this.sectors.Get(int32(this.header.headerDIFAT[i])); err != nil {
			return
		}
		if err = s.read(this.f); err != nil {
			return
		}
		if err = this.memory.addSector(s, MemoryTableFat); err != nil {
			return
		}
	}

	buf := make([]byte, this.SectorSize())
	//The last value is the offset so -1
	sz := this.SectorSize()/UInt32Size - 1

	if int32(this.header.numDIFATSector) > 0 {
		offset := int32(this.header.firstDIFATSectorLocation)
		for i := 0; i < int(this.header.numDIFATSector) && offset >= 0; i++ {
			if s, err = this.sectors.Get(offset); err != nil {
				return
			}
			err = s.Read(this.f, 0, buf)
			if err != nil {
				err = fmt.Errorf("Error read DIFAT sector %v: %v", s.id, err)
				return
			}
			if err = this.memory.addSector(s, MemoryDIFAT); err != nil {
				return
			}
			//---------------
			for j := 0; j < sz; j++ {
				SecID := int32(ParseUint32(buf[j*UInt32Size : j*UInt32Size+UInt32Size]))
				if SecID < 0 || int(SecID) >= this.sectors.Len() {
					break
				}
				if s, err = this.sectors.Get(SecID); err != nil {
					return
				}
				if err = s.read(this.f); err != nil {
					return
				}
				if err = this.memory.addSector(s, MemoryTableFat); err != nil {
					return
				}

			}
			offset = int32(ParseUint32(buf[len(buf)-UInt32Size:]))
		}
	}

	idx := 0
	it := this.memory.NewUInt32Iterator(MemoryTableFat)
	for it.Next() && idx < this.sectors.Len() {
		s, _ := this.sectors.Get(int32(idx))
		s.setNext(it.Value())
		if s.next == FREESECT {
			if err = this.memory.Push(s); err != nil {
				return
			}
		}
		idx++
	}
	return
}

func (this *CompoundFile) readDirectory() (err error) {
	var s *Sector

	c := 10
	if this.header.numDirectorySector > 0 {
		c = int(this.header.numDirectorySector)
	}
	this.directory = newDirectoryCollection(c)

	cycles := make(map[*Sector]bool)
	sz := this.header.DirEntry()
	off := int32(this.header.firstDirectorySectorLocation)

	buf := make([]byte, this.SectorSize())
	for off >= 0 {
		if s, err = this.sectors.Get(int32(off)); err != nil {
			return fmt.Errorf("Directory entries read error: %v", err)
		}

		if cycles[s] {
			return fmt.Errorf("directory entries sector cycle: %v", off)
		} else {
			cycles[s] = true
		}

		err := s.Read(this.f, 0, buf)
		if err != nil {
			return fmt.Errorf("Directory entries read error: %v", err)
		}
		if err = this.memory.addSector(s, MemoryDir); err != nil {
			return fmt.Errorf("Directory error: %v", err)
		}
		//-------------------------------
		for j := 0; j < int(sz); j++ {
			de := NewDirectory()
			if err = de.read(buf[j*DirectorySize : j*DirectorySize+DirectorySize]); err != nil {
				return err
			}
			if err = this.directory.Add(de); err != nil {
				return err
			}
			if de.objectType == StgUnallocated {
				if err = this.directory.Push(de); err != nil {
					return err
				}
			}
		}
		off = int32(s.next)
		if off < 0 && uint32(off) != ENDOFCHAIN {
			return fmt.Errorf("directory entries error finding sector: %v", off)
		}
	}
	return
}

func (this *CompoundFile) RootStorage() *Storage {
	if this == nil {
		return nil
	}
	return this.root
}

func (this *CompoundFile) readMiniFAT() (err error) {
	c := int32(this.header.numMiniFATSector)
	SecID := int32(this.header.firstMiniFATSectorLocation)
	var s *Sector
	for i := 0; i < int(c) && SecID >= 0; i++ {
		if s, err = this.sectors.Get(SecID); err != nil {
			err = fmt.Errorf("Get MiniFAT error: %v", err)
			return
		}
		if err = s.read(this.f); err != nil {
			err = fmt.Errorf("MiniFAT read error: %v", err)
			return
		}
		if err = this.memory.addSector(s, MemoryTableMini); err != nil {
			return
		}
		//----------------------------------
		SecID = int32(s.next)
	}

	///-------------- Mini FAT sectors ---------------
	if this.directory.Len() > 0 {
		var de *Directory
		var s *Sector
		var mini *MiniSector

		if de, err = this.directory.Get(0); err != nil {
			return
		}
		SecID := int32(de.startSectorLocation)
		sz := this.SectorSize() / this.MiniSectorSize()
		for SecID >= 0 {
			if s, err = this.sectors.Get(SecID); err != nil {
				return
			}
			if this.memory.FindSector(s) {
				err = fmt.Errorf("Error adding sector because sector data take up memory: %v", s)
				return
			}
			s.sectorType = TypeSectorMiniFAT
			//----------------------------------
			for i := 0; i < sz; i++ {
				mini = newMiniSector(this.MiniSectorSize(), i*this.MiniSectorSize(), s)
				if err = this.mini.addSector(mini); err != nil {
					return
				}
			}
			SecID = int32(s.next)
		}

		idx := 0
		it := this.memory.NewUInt32Iterator(MemoryTableMini)
		for it.Next() && idx < this.mini.Len() {
			mini, _ = this.mini.Get(idx)
			mini.next = it.Value()
			if mini.next == FREESECT {
				err = this.mini.Push(mini)
				if err != nil {
					return err
				}
			}
			idx++
		}
	}
	return
}

func (this *CompoundFile) Version() int {
	if this == nil || this.header == nil {
		return -1
	}
	return this.header.getVersion()
}

func (this *CompoundFile) Close() {
	if this == nil {
		return
	}
	this.header = nil
	if this.f != nil {
		_ = this.f.Close()
	}

	//memory
	if this.memory != nil {
		this.memory.Close()
		this.memory = nil
	}

	//Sectors
	if this.sectors != nil {
		this.sectors.Close()
		this.sectors = nil
	}
	if this.mini != nil {
		this.mini.Close()
		this.mini = nil
	}

	//Directory
	this.directory.Close()
	this.directory = nil
}

func (this *CompoundFile) updateDirectory(de *Directory) (err error) {
	var s *Sector
	if de.id < 0 {
		err = fmt.Errorf("Error add directory: id = %v", de.id)
		return
	}

	count := this.SectorSize() / DirectorySize
	index := de.id / count
	offset := de.id % count
	if index >= this.memory.Len(MemoryDir) {
		err = fmt.Errorf("No memory allocated for the directory: %v", de)
		//if s, err = this.addSector(TypeSectorMemmoryDirectory); err != nil {
		//	return
		//}
		return
	} else {
		if s, err = this.memory.getSector(MemoryDir, int32(index)); err != nil {
			return
		}
	}
	err = s.Write(offset*DirectorySize, de)
	s.modified = true
	return
}

func (this *CompoundFile) addMiniSector() (*MiniSector, error) {
	//-----------
	s := this.mini.Pop()
	if s != nil {
		return s, nil
	}
	if _, err := this.addSector(TypeSectorMiniFAT); err != nil {
		return nil, err
	}
	s = this.mini.Pop()
	if s != nil {
		return s, nil
	}
	return nil, fmt.Errorf("Eror add mini sector fat")
}

func (this *CompoundFile) addSector(Type SectorType) (*Sector, error) {
	switch Type {
	case TypeSectorMemmoryDirectory:
		old := this.memory.getLastSector(MemoryDir)
		s, err := this.addSector(TypeSectorFAT)
		if err != nil {
			return nil, err
		}
		s.modified = true
		if err = this.memory.addSector(s, MemoryDir); err != nil {
			return nil, err
		}

		if err = this.memory.changeFAT(s); err != nil {
			return nil, err
		}
		if old != nil {
			old.next = uint32(s.id)
			if err = this.memory.changeFAT(old); err != nil {
				return nil, err
			}
		}
		count := this.SectorSize() / DirectorySize
		for i := 0; i < count; i++ {
			empty := NewDirectory()
			if err = this.directory.Add(empty); err != nil {
				return nil, err
			}
			if err = this.directory.Push(empty); err != nil {
				return nil, err
			}
			if err = s.Write(i*DirectorySize, empty); err != nil {
				return nil, err
			}
		}

		if int32(this.header.firstDirectorySectorLocation) < 0 {
			fst, err := this.memory.getSector(MemoryDir, 0)
			if err != nil {
				return nil, err
			}
			this.header.firstDirectorySectorLocation = uint32(fst.id)
			this.header.modified = true
		}
		return s, nil
	case TypeSectorFAT:
		var s *Sector
		var err error

		if this.memory.Len(MemoryFree) > 0 {
			if s, err = this.memory.Pop(); err != nil {
				return nil, err
			}
			return s, nil
		}
		if this.sectors.Len() >= this.memory.CountUint32(MemoryTableFat) {
			if _, err = this.addSector(TypeSectorMemmoryFAT); err != nil {
				return nil, err
			}
		}
		s = newSector(this.SectorSize())
		s.modified = true
		s.data = make([]byte, s.size)
		this.sectors.Add(s)
		if err = this.memory.changeFAT(s); err != nil {
			return nil, err
		}
		return s, nil
	case TypeSectorMemmoryFAT:
		sz := this.SectorSize() / UInt32Size

		// Add table FAT
		s := newSector(this.SectorSize())
		s.data = make([]byte, s.size)
		for i := 0; i < sz; i++ {
			if err := s.Write(i*UInt32Size, FREESECT); err != nil {
				return nil, err
			}
		}
		this.sectors.Add(s)
		if err := this.memory.addSector(s, MemoryTableFat); err != nil {
			return nil, err
		}

		if err := this.memory.changeFAT(s); err != nil {
			return nil, err
		}

		//Add in header or DIFFAT
		size := this.memory.Len(MemoryTableFat)
		if size <= len(this.header.headerDIFAT) &&
			this.header.numDIFATSector == 0 {
			//size <= 109

			this.header.numFATSector = uint32(size)
			this.header.headerDIFAT[this.header.numFATSector-1] = uint32(s.id)
		} else {
			//size >= 110
			var difat *Sector

			size := size - len(this.header.headerDIFAT) - 1
			//last value - offset
			index := size / (sz - 1)
			offset := size % (sz - 1)
			if index >= this.memory.Len(MemoryDIFAT) {
				old := this.memory.getLastSector(MemoryDIFAT)
				difat = newSector(this.SectorSize())
				difat.data = make([]byte, difat.size)
				//Write sector ID
				if err := difat.Write(0, uint32(s.id)); err != nil {
					return nil, err
				}
				//Write free
				for i := 1; i < (sz - 1); i++ {
					if err := difat.Write(i*UInt32Size, FREESECT); err != nil {
						return nil, err
					}
				}
				//Write last ENDOFCHAIN
				if err := difat.Write(difat.size-UInt32Size, ENDOFCHAIN); err != nil {
					return nil, err
				}
				this.sectors.Add(difat)

				if old != nil {
					if err := old.Write(old.size-UInt32Size, uint32(difat.id)); err != nil {
						return nil, err
					}
				} else {
					this.header.firstDIFATSectorLocation = uint32(difat.id)
				}

				if err := this.memory.addSector(difat, MemoryDIFAT); err != nil {
					return nil, err
				}
				this.header.numDIFATSector = uint32(this.memory.Len(MemoryDIFAT))

				if err := this.memory.changeFAT(difat); err != nil {
					return nil, err
				}
			} else {
				var err error
				difat, err = this.memory.getSector(MemoryDIFAT, int32(index))
				if err != nil {
					return nil, err
				}
				if err = difat.Write(offset*UInt32Size, uint32(s.id)); err != nil {
					return nil, err
				}
			}
		}
		return s, nil
	case TypeSectorMiniFAT:
		if this.memory.CountUint32(MemoryTableMini) <= this.mini.Len() {
			_, err := this.addSector(TypeSectorMemmoryMiniFAT)
			if err != nil {
				return nil, err
			}
		}
		old := this.mini.getLastSector()
		s, err := this.addSector(TypeSectorFAT)
		if err != nil {
			return nil, err
		}
		s.next = ENDOFCHAIN
		s.sectorType = TypeSectorMiniFAT
		s.modified = true
		if err = this.memory.changeFAT(s); err != nil {
			return nil, err
		}

		if old == nil {
			de, err := this.directory.Get(0)
			if err != nil {
				return nil, err
			}
			de.startSectorLocation = uint32(s.id)
			if err = this.updateDirectory(de); err != nil {
				return nil, err
			}
		} else {
			old.sector.next = uint32(s.id)
			if err = this.memory.changeFAT(old.sector); err != nil {
				return nil, err
			}
		}

		sz := this.sectorSize / this.MiniSectorSize()
		for i := 0; i < sz; i++ {
			mini := newMiniSector(this.MiniSectorSize(), i*this.MiniSectorSize(), s)
			if err = this.mini.addSector(mini); err != nil {
				return nil, err
			}
			if err = this.mini.Push(mini); err != nil {
				return nil, err
			}
			if err = this.memory.changeMiniFAT(mini); err != nil {
				return nil, err
			}
		}
		return s, nil
	case TypeSectorMemmoryMiniFAT:
		s, err := this.addSector(TypeSectorFAT)
		if err != nil {
			return nil, err
		}

		old := this.memory.getLastSector(MemoryTableMini)
		if old == nil {
			this.header.firstMiniFATSectorLocation = uint32(s.id)
			this.header.modified = true
		} else {
			old.next = uint32(s.id)
			if err = this.memory.changeFAT(old); err != nil {
				return nil, err
			}
		}

		if err = this.memory.addSector(s, MemoryTableMini); err != nil {
			return nil, err
		}
		this.header.numMiniFATSector = uint32(this.memory.Len(MemoryTableMini))
		this.header.modified = true

		if err = this.memory.changeFAT(s); err != nil {
			return nil, err
		}
		sz := this.SectorSize() / UInt32Size
		for i := 0; i < sz; i++ {
			if err = s.Write(i*UInt32Size, FREESECT); err != nil {
				return nil, err
			}
		}
		return s, nil
	}
	return nil, fmt.Errorf("unknown type sector: %v", Type)
}

func (this *CompoundFile) Save(filename string) error {
	if this == nil {
		return fmt.Errorf("The file is not saved: %v", filename)
	}

	b, err := this.header.Bytes()
	if err != nil {
		return err
	}
	f := NewFile()
	defer f.Close()

	if err = f.WriteAt(b, 0); err != nil {
		return err
	}
	it := this.sectors.Iterator()
	offset := HeaderSize
	b = make([]byte, this.SectorSize())
	for it.Next() {
		if err = it.Value().Read(this.f, 0, b); err != nil {
			return err
		}
		if err = f.WriteAt(b, offset); err != nil {
			return err
		}
		offset += this.SectorSize()
	}
	return f.Save(filename, 0644)
}

func (this *CompoundFile) Commit() error {
	if this == nil || this.f == nil {
		return fmt.Errorf("The file is not saved")
	}

	if this.header.modified {
		b, err := this.header.Bytes()
		if err != nil {
			return err
		}
		if _, err = this.f.WriteAt(b, 0); err != nil {
			return err
		}
		this.header.modified = false
	}

	it := this.sectors.Iterator()
	offset := int64(HeaderSize)
	b := make([]byte, this.SectorSize())
	for it.Next() {
		s := it.Value()
		if !s.modified {
			continue
		}
		err := s.Read(this.f, 0, b)
		if err != nil {
			return err
		}
		if _, err = this.f.WriteAt(b, offset); err != nil {
			return err
		}
		s.modified = false
		offset += int64(this.SectorSize())
	}
	return nil
}

func (this *CompoundFile) FreeFAT(SecID int32, size int) (err error) {
	if SecID < 0 || size <= 0 {
		err = fmt.Errorf("Error free FAT: %v - %v", SecID, size)
		return
	}
	offset := 0
	var s *Sector
	for offset < size && SecID >= 0 {
		if s, err = this.sectors.Get(SecID); err != nil {
			return
		}
		SecID = int32(s.next)
		if s.data == nil {
			s.data = make([]byte, s.size)
			s.modified = true
		}

		if err = this.memory.Push(s); err != nil {
			return
		}
		if err = this.memory.changeFAT(s); err != nil {
			return
		}
	}
	return
}

func (this *CompoundFile) FreeMiniFAT(SecID int32, size int) (err error) {
	if SecID < 0 || size <= 0 {
		err = fmt.Errorf("Error free mini FAT: %v - %v", SecID, size)
		return
	}
	offset := 0
	var s *MiniSector
	for offset < size && SecID >= 0 {
		if s, err = this.mini.Get(int(SecID)); err != nil {
			return
		}
		SecID = int32(s.next)
		if err = this.mini.Push(s); err != nil {
			return
		}
		if err = this.memory.changeMiniFAT(s); err != nil {
			return
		}
	}
	return
}

func (this *CompoundFile) SectorBytes(SecID int32) (b []byte, err error) {
	var s *Sector
	if s, err = this.sectors.Get(SecID); err != nil {
		return
	}
	b = make([]byte, s.size)
	err = s.Read(this.f, 0, b)
	return
}
