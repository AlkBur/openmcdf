package openmcdf

import (
	"errors"
	"time"
)

var StreamNotFound = errors.New("Storage not found")

type Stream struct {
	cf *CompoundFile
	de *Directory
}

func newBaseStream(de *Directory, cf *CompoundFile) *Stream {
	return &Stream{de: de, cf: cf}
}

func newStream(name string, cf *CompoundFile) *Stream {
	de := newDirectory()
	de.setObjectType(StgStream)
	if err := de.setName(name); err != nil {
		return nil
	}
	t := time.Now()
	de.newGUID()
	de.setTimeCreate(t)
	de.setTimeModification(t)

	return newBaseStream(de, cf)
}

func (this *Stream) GetData() (b []byte, err error) {
	if this == nil {
		return nil, errors.New("Stream is null")
	} else if this.de == nil {
		return nil, errors.New("Directory is null")
	}
	b, err = this.de.Read(this.cf)
	return
}

func (this *Stream) String() string {
	return this.de.String()
}

func (this *Stream) Size() int64 {
	if this == nil {
		return 0
	}
	return int64(this.de.size)
}

func (this *Stream) SetData(b []byte) (err error) {
	if this == nil {
		err = errors.New("Error set data: stream is nil")
		return
	}
	if this.de == nil {
		err = errors.New("Error set data: directory is nil")
		return
	}

	err = this.de.Write(this.cf, b)
	return
}
