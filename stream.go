package openmcdf

import (
	"errors"
)

var StreamNotFound = errors.New("Stream not found")

type Stream struct {
	cf *CompoundFile
	de *Directory
}

func newStream(de *Directory, cf *CompoundFile) *Stream {
	return &Stream{de: de, cf: cf}
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

func (this *Stream) Append(b []byte) (err error) {
	buf, err := this.GetData()
	if err != nil {
		return
	}
	buf = append(buf, b...)

	err = this.de.Write(this.cf, buf)
	return
}
