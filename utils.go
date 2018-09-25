package openmcdf

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"
)

type File struct {
	data []byte
}

func NewFile() *File {
	return &File{
		data: make([]byte, 512),
	}
}

func (this *File) Open(filename string) (err error) {
	this.data, err = ioutil.ReadFile(filename)
	return
}

func (this *File) Hash() string {
	h := md5.New()
	buf := bytes.NewBuffer(this.data)
	if _, err := io.Copy(h, buf); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (this *File) Save(file string, perm os.FileMode) error {
	return ioutil.WriteFile(file, this.data, perm)
}

func (this *File) Size() int {
	return len(this.data)
}

func (this *File) Close() error {
	this.data = nil
	return nil
}

func (this *File) ReadAt(off, n int) []byte {
	return this.data[off : off+n]
}

func (this *File) WriteAt(b []byte, off int) (err error) {
	if off < 0 {
		err = io.EOF
		return
	}
	n := len(b)
	if off+n > this.Size() {
		add := off + n - this.Size()
		tmp := make([]byte, add)
		this.data = append(this.data, tmp...)

		if this.Size() != len(b)+off {
			err = fmt.Errorf("Error write: size %d != %d", this.Size(), len(b)+off)
			return
		}
	}
	n = copy(this.data[off:], b)
	if n < len(b) {
		err = fmt.Errorf("Write %d less than you need %d", n, len(b))
		return
	}
	return
}

func (this *File) Reader() io.Reader {
	return bytes.NewBuffer(this.data)
}

///////////////////////////////////////////////

func ParseUint32(b []byte) uint32 {
	return binary.LittleEndian.Uint32(b)
}

func ReadData(r io.Reader, data interface{}) error {
	return binary.Read(r, binary.LittleEndian, data)
}

func WriteData(w io.Writer, data interface{}) error {
	return binary.Write(w, binary.LittleEndian, data)
}

///////////////////////////////////////////////

func RecoverError(err error) {
	if r := recover(); r != nil {
		var ok bool
		err, ok = r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}
	}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

///////////////////////////////////////////////

func ToTime(t uint64) time.Time {
	//Seconds
	t1 := int64(t / 10000000)
	//Fractional amount of a second
	t2 := int64(t % 10000000)
	return time.Unix(t1, t2)
}

func toTimestamp(t time.Time) uint64 {
	return uint64(t.UTC().Unix())
}
