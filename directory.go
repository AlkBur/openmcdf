package openmcdf

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf16"
)

const DirectorySize = 128

const (
	Red   = 0
	Black = 1

	NOSTREAM = uint32(0xFFFFFFFF) //-1
)

const (
	StgUnallocated = 0
	StgStorage     = 1
	StgStream      = 2
	StgRoot        = 5
)

type DirectoryCollection struct {
	data []*Directory
	free map[*Directory]bool
}

type DirectoryIterator struct {
	current int32
	data    *DirectoryCollection
}

type Directory struct {
	name                [32]uint16
	nameLen             uint16
	objectType          uint8
	colorFlag           uint8
	leftSiblingID       uint32 // Note that it's actually the left/right child in the RB-tree.
	rightSiblingID      uint32 // So entry.leftSibling.rightSibling does NOT go back to entry.
	childID             uint32
	clsid               [16]byte
	stateBits           uint32
	creationTime        uint64
	modifiedTime        uint64
	startSectorLocation uint32
	size                uint64
	//---------
	id int
}

//--------------Directory collection--------------

func newDirectoryCollection(cap int) *DirectoryCollection {
	return &DirectoryCollection{
		data: make([]*Directory, 0, cap),
		free: make(map[*Directory]bool),
	}
}

func (this *DirectoryCollection) Len() int {
	return len(this.data)
}

func (this *DirectoryCollection) Add(de *Directory) (err error) {
	if de.id >= 0 {
		err = fmt.Errorf("Directory has already been added: %v", de)
		return
	}
	de.id = this.Len()
	this.data = append(this.data, de)
	return
}

func (this *DirectoryCollection) check(de *Directory) (err error) {
	id := de.id
	if id < 0 || id >= this.Len() || de != this.data[id] {
		err = fmt.Errorf("Error ID directory: %v", de)
	}
	return
}

func (this *DirectoryCollection) Delete(de *Directory) (err error) {
	if err = this.check(de); err != nil {
		return
	}
	//delete element
	delete(this.free, de)
	this.data = append(this.data[:de.id], this.data[de.id+1:]...)
	return
}

func (this *DirectoryCollection) Get(id int32) (de *Directory, err error) {
	if id < 0 || int(id) >= this.Len() {
		err = fmt.Errorf("Directory index out of range: %v", id)
		return
	}
	de = this.data[id]
	return
}

func (this *DirectoryCollection) getLeft(de *Directory) *Directory {
	if de.leftSiblingID == NOSTREAM {
		return nil
	}
	if int32(de.leftSiblingID) < 0 || int(de.leftSiblingID) >= this.Len() {
		return nil
	}
	return this.data[de.leftSiblingID]
}

func (this *DirectoryCollection) getRight(de *Directory) *Directory {
	if de.rightSiblingID == NOSTREAM {
		return nil
	}
	if int32(de.rightSiblingID) < 0 || int(de.rightSiblingID) >= this.Len() {
		return nil
	}
	return this.data[de.rightSiblingID]
}

func (this *DirectoryCollection) getChild(de *Directory) *Directory {
	if de == nil || de.childID == NOSTREAM {
		return nil
	}
	if int32(de.childID) < 0 || int(de.childID) >= this.Len() {
		return nil
	}
	return this.data[de.childID]
}

func (this *DirectoryCollection) Close() {
	if this == nil || this.data == nil {
		return
	}
	for i := range this.data {
		this.data[i] = nil
	}
	for de := range this.free {
		delete(this.free, de)
	}
	this.data = nil
	this.free = nil
}

func (this *DirectoryCollection) Iterator() *DirectoryIterator {
	return &DirectoryIterator{current: -1, data: this}
}

func (this *DirectoryIterator) Value() *Directory {
	de, err := this.data.Get(this.current)
	if err != nil {
		return nil
	}
	return de
}
func (this *DirectoryIterator) Next() bool {
	this.current++
	if this.current >= int32(this.data.Len()) {
		return false
	}
	return true
}

func (this *DirectoryCollection) String() string {
	str := strings.Builder{}
	str.WriteString("Directories: [")
	for _, de := range this.data {
		str.WriteString("\n\t")
		str.WriteString(de.String())
	}
	if len(this.data) > 0 {
		str.WriteString("\n")
	}
	str.WriteString("]")
	return str.String()
}

func (this *DirectoryCollection) New(cf *CompoundFile, name string, objectType uint8) (*Directory, error) {
	de := this.Pop()
	if de == nil {
		count := cf.SectorSize() / DirectorySize
		if this.Len()/count >= cf.memory.Len(MemoryDir) {
			if _, err := cf.addSector(TypeSectorMemmoryDirectory); err != nil {
				return nil, err
			}
		}
		de = this.Pop()
		if de == nil {
			return nil, fmt.Errorf("Error allocated directory memory")
		}
	}
	t := time.Now()
	err := de.SetName(name)
	de.objectType = objectType
	de.newGUID()
	de.colorFlag = Black
	de.setTimeCreate(t)
	de.setTimeModification(t)

	return de, err
}

func (this *DirectoryCollection) Pop() *Directory {
	if len(this.free) == 0 {
		return nil
	}
	key := make([]*Directory, len(this.free))
	i := 0
	for k := range this.free {
		key[i] = k
		i++
	}
	if len(key) > 1 {
		sort.Slice(key, func(i, j int) bool {
			return key[i].id < key[j].id
		})
	}
	de := key[0]
	delete(this.free, de)
	return de
}

func (this *DirectoryCollection) Push(de *Directory) (err error) {
	if err = this.check(de); err != nil {
		return
	}
	_, ok := this.free[de]
	if ok {
		err = fmt.Errorf("Directory already added in free: %v", de)
		return
	}
	de.clear()
	this.free[de] = true
	return
}

//--------------Directory--------------

func NewDirectory() *Directory {
	this := &Directory{id: -1}
	this.clear()
	return this
}

func (this *Directory) clear() {
	for i := 0; i < len(this.name); i++ {
		this.name[i] = 0
	}
	this.nameLen = 0
	this.objectType = StgUnallocated
	this.colorFlag = Red
	this.leftSiblingID = NOSTREAM
	this.rightSiblingID = NOSTREAM
	this.childID = NOSTREAM
	for i := 0; i < len(this.clsid); i++ {
		this.clsid[i] = 0
	}
	this.stateBits = 0
	this.creationTime = 0
	this.modifiedTime = 0
	this.startSectorLocation = ENDOFCHAIN
	this.size = 0
}

func (this *Directory) setObjectType(objectType uint8) error {
	if objectType > 2 && objectType != StgRoot {
		return fmt.Errorf("Error set object type: %v", objectType)
	}
	this.objectType = objectType
	return nil
}

func (this *Directory) setTimeCreate(t time.Time) {
	this.creationTime = toTimestamp(t)
}

func (this *Directory) setTimeModification(t time.Time) {
	this.modifiedTime = toTimestamp(t)
}

func (this *Directory) getTimeCreate() time.Time {
	return ToTime(this.creationTime)
}

func (this *Directory) getTimeModification() (t time.Time) {
	return ToTime(this.modifiedTime)
}

func (this *Directory) SetName(name string) error {
	if strings.Index(name, "\\") >= 0 ||
		strings.Index(name, "/") >= 0 ||
		strings.Index(name, ":") >= 0 ||
		strings.Index(name, "!") >= 0 {
		return fmt.Errorf("Invalid set directory name: %s", name)
	}

	if len(name) > 31 {
		return fmt.Errorf("Invalid len directory name: %v", len(name))
	}
	temp := utf16.Encode([]rune(name))
	//var newName []byte
	//newName = *(*[]byte)(unsafe.Pointer(&temp))
	copy(this.name[:], temp)

	this.name[len(temp)] = 0x0000
	this.nameLen = uint16((len(temp) + 1) * 2)

	return nil
}

func (this *Directory) String() string {
	return fmt.Sprintf(`Directory{
	ID: %d
	Name: %s
	Name bytes: %v
	Size name: %v
	Type: %v
	Color: %v
	Left sibling ID: %v
	Right sibling ID: %v
	Child ID: %v
	ID: %v
	State bits: %v
	Start sector location: %v
	Size: %v
}`, this.id, this.Name(), this.name, this.nameLen, this.objectType,
		this.colorFlag, this.leftSiblingID, this.rightSiblingID, this.childID,
		this.clsid, this.stateBits, this.startSectorLocation, this.size,
	)
}

func (this *Directory) Name() string {
	name := ""
	if this.nameLen > 0 {
		nlen := int(this.nameLen/2 - 1)
		if nlen > 32 {
			nlen = 31
		}

		slen := 0
		//if !unicode.IsPrint(rune(this.name[0])) {
		//	slen = 1
		//}
		name = string(utf16.Decode(this.name[slen:nlen]))
	}
	return name
}

func (this *Directory) read(b []byte) (err error) {
	defer RecoverError(err)

	r := bytes.NewBuffer(b)
	//Read 128 bytes
	check(ReadData(r, this.name[:]))              //64 byte
	check(ReadData(r, &this.nameLen))             //2 byte
	check(ReadData(r, &this.objectType))          //1 byte
	check(ReadData(r, &this.colorFlag))           //1 byte
	check(ReadData(r, &this.leftSiblingID))       //4 byte
	check(ReadData(r, &this.rightSiblingID))      //4 byte
	check(ReadData(r, &this.childID))             //4 byte
	check(ReadData(r, this.clsid[:]))             //16 byte
	check(ReadData(r, &this.stateBits))           //4 byte
	check(ReadData(r, this.creationTime))         //8 byte
	check(ReadData(r, this.modifiedTime))         //8 byte
	check(ReadData(r, &this.startSectorLocation)) //4 byte
	check(ReadData(r, &this.size))                //8 byte

	return
}

func (this *Directory) compareTo(otherDir *Directory) int {
	return strings.Compare(this.Name(), otherDir.Name())
}

func (this *Directory) newGUID() {
	rand.Read(this.clsid[:])
	//Set version 4
	this.clsid[6] = (this.clsid[6] & 0x0f) | (4 << 4)
	//Set variant Microsoft
	this.clsid[8] = (this.clsid[8]&(0xff>>3) | (0x06 << 5))
}

func (this *Directory) newStream(cf *CompoundFile) *Stream {
	if this == nil || this.objectType != StgStream {
		return nil
	}
	return newStream(this, cf)
}

func (this *Directory) newStorage(cf *CompoundFile) *Storage {
	if this == nil || this.objectType != StgStorage {
		return nil
	}
	return newStorage(this, cf)
}

func (this *Directory) newRootStorage(cf *CompoundFile) *Storage {
	if this == nil || this.objectType != StgRoot {
		return nil
	}
	return newStorage(this, cf)
}

func (this *Directory) Bytes() (b []byte, err error) {
	defer RecoverError(err)

	buf := new(bytes.Buffer)

	check(WriteData(buf, this.name[:]))
	//32*2=64
	check(WriteData(buf, this.nameLen))
	//66
	check(WriteData(buf, this.objectType))
	//67
	check(WriteData(buf, this.colorFlag))
	//68
	check(WriteData(buf, this.leftSiblingID))
	//72
	check(WriteData(buf, this.rightSiblingID))
	//76
	check(WriteData(buf, this.childID))
	//76+4=80
	check(WriteData(buf, this.clsid[:]))
	//80+16=96
	check(WriteData(buf, this.stateBits))
	//100
	check(WriteData(buf, this.creationTime))
	//108
	check(WriteData(buf, this.modifiedTime))
	//116
	check(WriteData(buf, this.startSectorLocation))
	//120
	check(WriteData(buf, this.size))
	//128

	b = buf.Bytes()
	return
}

func (this *Directory) Write(cf *CompoundFile, b []byte) (err error) {
	if this == nil {
		err = errors.New("Error write in directory: directory is nil")
		return
	}
	if b == nil {
		err = errors.New("Error write in directory: data is nil")
		return
	}

	OldSize := int(this.size)
	NewSize := len(b)

	if OldSize >= int(cf.header.miniStreamCutoffSize) &&
		NewSize < int(cf.header.miniStreamCutoffSize) {
		//Clear sectory FAT
		if err = cf.FreeFAT(int32(this.startSectorLocation), OldSize); err != nil {
			return
		}
		OldSize = 0
	} else if (OldSize > 0 && OldSize < int(cf.header.miniStreamCutoffSize) &&
		NewSize >= int(cf.header.miniStreamCutoffSize)) ||
		(OldSize > 0 && NewSize == 0) {
		//Clear mini sectory FAT
		if err = cf.FreeMiniFAT(int32(this.startSectorLocation), OldSize); err != nil {
			return
		}
		OldSize = 0
	}

	//Change directory
	updateDE := false
	if OldSize != NewSize {
		this.size = uint64(NewSize)
		updateDE = true
	}

	//Write data
	if NewSize <= 0 {
		this.startSectorLocation = ENDOFCHAIN
		updateDE = true
	} else {
		offset := 0
		if NewSize >= int(cf.header.miniStreamCutoffSize) {
			//sector FAT
			var s, old *Sector
			SecID := int32(this.startSectorLocation)
			for offset < NewSize {
				if OldSize > offset && SecID >= 0 {
					if s, err = cf.sectors.Get(SecID); err != nil {
						return
					}
					if s.data == nil {
						s.data = make([]byte, s.size)
					}
				} else {
					if s, err = cf.addSector(TypeSectorFAT); err != nil {
						return
					}
					s.next = ENDOFCHAIN
					if old != nil {
						old.next = uint32(s.id)
						if err = cf.memory.changeFAT(old); err != nil {
							return
						}
					} else {
						this.startSectorLocation = uint32(s.id)
						updateDE = true
					}
				}
				if err = s.Write(0, b[offset:]); err != nil {
					return
				}
				offset += s.size
				old = s
				SecID = int32(s.next)
			}
			if err = cf.memory.changeFAT(s); err != nil {
				return
			}
			//clear
			SecID = int32(s.next)
			if SecID > 0 && offset < OldSize {
				if err = cf.FreeFAT(SecID, OldSize-offset); err != nil {
					return
				}
			}
		} else {
			//mini sector FAT
			var s, old *MiniSector
			SecID := int32(this.startSectorLocation)
			for offset < NewSize {
				if OldSize > offset && SecID >= 0 {
					if s, err = cf.mini.Get(int(SecID)); err != nil {
						return
					}
				} else {
					if s, err = cf.addMiniSector(); err != nil {
						return
					}
					s.next = ENDOFCHAIN
					if old != nil {
						old.next = uint32(s.id)
						if err = cf.memory.changeMiniFAT(old); err != nil {
							return
						}
					} else {
						this.startSectorLocation = uint32(s.id)
						updateDE = true
					}
				}
				if s.sector.data == nil {
					if err = s.sector.read(cf.f); err != nil {
						return
					}
				}
				if err = s.Write(b[offset:]); err != nil {
					return
				}
				offset += s.size
				old = s
				SecID = int32(s.next)
			}
			if err = cf.memory.changeMiniFAT(s); err != nil {
				return
			}
			//clear
			SecID = int32(s.next)
			if SecID > 0 && offset < OldSize {
				if err = cf.FreeMiniFAT(SecID, OldSize-offset); err != nil {
					return
				}
			}
		}
	}
	if updateDE {
		if err = cf.updateDirectory(this); err != nil {
			return
		}
	}
	return
}

func (this *Directory) Read(cf *CompoundFile) (b []byte, err error) {
	if int64(this.size) <= 0 {
		return
	} else if int32(this.startSectorLocation) < 0 {
		err = fmt.Errorf("Start location is error in directory: %v", this.startSectorLocation)
		return
	}
	b = make([]byte, this.size)
	if len(b) < int(cf.header.miniStreamCutoffSize) {
		//Mini sector FAT
		var s *MiniSector

		//Start read
		SecID := int32(this.startSectorLocation)
		offset := 0
		for offset < len(b) {
			if s, err = cf.mini.Get(int(SecID)); err != nil {
				return
			}
			if err = s.Read(cf.f, b[offset:]); err != nil {
				return
			}
			offset += s.size
			SecID = int32(s.next)
		}
	} else {
		//Sector FAT
		var s *Sector
		SecID := int32(this.startSectorLocation)
		if SecID <= 0 {
			err = fmt.Errorf("Start location is error in directory: %v", SecID)
			return
		}
		offset := 0
		for offset < len(b) {
			s, err = cf.sectors.Get(SecID)
			if err != nil {
				return
			}
			if err = s.Read(cf.f, 0, b[offset:]); err != nil {
				return
			}
			offset += s.size
			SecID = int32(s.next)
		}
	}
	return
}
