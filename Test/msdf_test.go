package Test

import (
	"fmt"
	mcdf "github.com/AlkBur/openmcdf"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"
)

func Test_READ_STREAM(t *testing.T) {
	var find *mcdf.Stream
	const filename = "files/report.xls"

	cf, err := mcdf.Open(filename)
	defer cf.Close()
	assert.NoError(t, err)

	find, err = cf.RootStorage().GetStream("Workbook")
	assert.NoError(t, err)
	assert.NotNil(t, find)

	temp, err := find.GetData()
	assert.NoError(t, err)

	assert.NotNil(t, temp)
	assert.True(t, len(temp) > 0)
}

func Test_WRITE_STREAM(t *testing.T) {
	var myStream *mcdf.Stream
	const BUFFER_LENGTH = 10000
	b := GenBuffer(BUFFER_LENGTH)

	cf, err := mcdf.New(3)
	defer cf.Close()
	assert.NoError(t, err)

	myStream, err = cf.RootStorage().AddStream("MyStream")
	assert.NoError(t, err)
	assert.NotNil(t, myStream)

	assert.True(t, myStream.Size() == 0)
	err = myStream.SetData(b)
	assert.NoError(t, err)

	assert.True(t, myStream.Size() == BUFFER_LENGTH, "Stream size differs from buffer size")

	b2, err := myStream.GetData()
	assert.NoError(t, err)
	assert.True(t, len(b2) == BUFFER_LENGTH, "Stream size differs from buffer size")

	assert.Equal(t, b2, b)
}

func Test_WRITE_MINI_STREAM(t *testing.T) {
	var myStream *mcdf.Stream
	const BUFFER_LENGTH = 1023 // < 4096
	b := GenBuffer(BUFFER_LENGTH)

	cf, err := mcdf.New(3)
	defer cf.Close()
	assert.NoError(t, err)

	myStream, err = cf.RootStorage().AddStream("MyMiniStream")
	assert.NoError(t, err)
	assert.NotNil(t, myStream)
	assert.True(t, myStream.Size() == 0)

	err = myStream.SetData(b)
	assert.NoError(t, err)

	assert.True(t, (myStream.Size()) == BUFFER_LENGTH, "Mini Stream size differs from buffer size")

	b2, err := myStream.GetData()
	assert.NoError(t, err)
	assert.Equal(t, b, b2)
}

func Test_ZERO_LENGTH_WRITE_STREAM(t *testing.T) {
	var myStream *mcdf.Stream
	b := make([]byte, 0)
	const filename = "files/ZERO_LENGTH_STREAM.cfs"

	cf, err := mcdf.New(3)
	defer cf.Close()
	assert.NoError(t, err)

	myStream, err = cf.RootStorage().AddStream("MyStream")
	assert.NoError(t, err)
	assert.NotNil(t, myStream)
	assert.True(t, myStream.Size() == 0)

	err = myStream.SetData(b)
	assert.NoError(t, err)

	err = cf.Save(filename)
	assert.NoError(t, err)

	if _, err = os.Stat(filename); err == nil {
		err = os.Remove(filename)
		assert.NoError(t, err)
	}
}

func Test_ZERO_LENGTH_RE_WRITE_STREAM(t *testing.T) {
	var myStream *mcdf.Stream
	b := make([]byte, 0)
	const filename1 = "files/ZERO_LENGTH_STREAM_RE1.cfs"
	const filename2 = "files/ZERO_LENGTH_STREAM_RE2.cfs"

	{
		cf, err := mcdf.New(3)
		assert.NoError(t, err)

		myStream, err = cf.RootStorage().AddStream("MyStream")
		assert.NoError(t, err)
		assert.NotNil(t, myStream)
		assert.True(t, myStream.Size() == 0)

		err = myStream.SetData(b)
		assert.NoError(t, err)

		err = cf.Save(filename1)
		assert.NoError(t, err)
		cf.Close()
	}
	{
		cf, err := mcdf.Open(filename1)
		assert.NoError(t, err)

		myStream, err = cf.RootStorage().GetStream("MyStream")
		assert.NoError(t, err)
		assert.NotNil(t, myStream)
		assert.True(t, myStream.Size() == 0)

		err = myStream.SetData(make([]byte, 30))
		assert.NoError(t, err)
		err = cf.Save(filename2)
		assert.NoError(t, err)
		cf.Close()
	}

	if _, err := os.Stat(filename1); err == nil {
		err = os.Remove(filename1)
		assert.NoError(t, err)
	}
	if _, err := os.Stat(filename2); err == nil {
		err = os.Remove(filename2)
		assert.NoError(t, err)
	}
}

func Test_WRITE_STREAM_WITH_DIFAT(t *testing.T) {
	mcdf.SetLogger(t)

	var myStream *mcdf.Stream
	const filename = "files/WRITE_STREAM_WITH_DIFAT.cfs"

	// Incredible condition of 'resonance' between FAT and DIFAT sec number
	// 15345665 / 512 = 29972
	// 29972 / 128 = 234
	// FAT: 109; DIFAT: 234 - 109 = (125 + Derectory + AllocDIFAT) / 127 = 2
	const SIZE = 15345665
	b := GetBuffer(SIZE, 0)

	{
		cf, err := mcdf.New(3)
		assert.NoError(t, err)

		myStream, err = cf.RootStorage().AddStream("MyStream")
		assert.NoError(t, err)
		assert.NotNil(t, myStream)
		assert.True(t, myStream.Size() == 0)

		err = myStream.SetData(b)
		assert.NoError(t, err)

		err = cf.Save(filename)
		cf.Close()
	}

	{
		cf2, err := mcdf.Open(filename)
		assert.NoError(t, err)

		myStream, err = cf2.RootStorage().GetStream("MyStream")
		assert.NoError(t, err)
		assert.NotNil(t, myStream)
		assert.True(t, myStream.Size() == SIZE)

		tmp, err := myStream.GetData()
		assert.NoError(t, err)
		assert.True(t, len(tmp) == SIZE)
		assert.Equal(t, b, tmp)
		cf2.Close()
	}

	if _, err := os.Stat(filename); err == nil {
		err = os.Remove(filename)
		assert.NoError(t, err)
	}
}

func Test_WRITE_MINISTREAM_READ_REWRITE_STREAM(t *testing.T) {
	mcdf.SetLogger(t)

	const BIGGER_SIZE = 350
	const MEGA_SIZE = 18000000
	const filename1 = "files/WRITE_MINISTREAM_READ_REWRITE_STREAM.cfs"
	const filename2 = "files/WRITE_MINISTREAM_READ_REWRITE_STREAM_2ND.cfs"

	ba1 := GetBuffer(BIGGER_SIZE, 1)
	ba2 := GetBuffer(BIGGER_SIZE, 2)
	ba3 := GetBuffer(BIGGER_SIZE, 3)
	ba4 := GetBuffer(BIGGER_SIZE, 4)
	ba5 := GetBuffer(BIGGER_SIZE, 5)

	//WRITE 5 (mini)streams in a compound file --

	cfa, err := mcdf.New(3)
	assert.NoError(t, err)

	myStream, err := cfa.RootStorage().AddStream("MyFirstStream")
	assert.NoError(t, err)
	assert.NotNil(t, myStream)

	err = myStream.SetData(ba1)
	assert.NoError(t, err)

	assert.True(t, myStream.Size() == BIGGER_SIZE)

	myStream2, err := cfa.RootStorage().AddStream("MySecondStream")
	assert.NoError(t, err)
	assert.NotNil(t, myStream2)

	err = myStream2.SetData(ba2)
	assert.True(t, myStream2.Size() == BIGGER_SIZE)

	myStream3, err := cfa.RootStorage().AddStream("MyThirdStream")
	assert.NoError(t, err)
	assert.NotNil(t, myStream3)

	err = myStream3.SetData(ba3)
	assert.True(t, myStream3.Size() == BIGGER_SIZE)

	myStream4, err := cfa.RootStorage().AddStream("MyFourthStream")
	assert.NoError(t, err)
	assert.NotNil(t, myStream4)

	err = myStream4.SetData(ba4)
	assert.True(t, myStream4.Size() == BIGGER_SIZE)

	myStream5, err := cfa.RootStorage().AddStream("MyFifthStream")
	assert.NoError(t, err)
	assert.NotNil(t, myStream5)

	err = myStream5.SetData(ba5)
	assert.True(t, myStream5.Size() == BIGGER_SIZE)

	err = cfa.Save(filename1)

	cfa.Close()

	// Now get the second stream and rewrite it smaller
	bb := GenBuffer(MEGA_SIZE)
	cfb, err := mcdf.Open(filename1)
	assert.NoError(t, err)

	myStreamB, err := cfb.RootStorage().GetStream("MySecondStream")
	assert.NoError(t, err)
	assert.NotNil(t, myStreamB)

	err = myStreamB.SetData(bb)
	assert.NoError(t, err)
	assert.True(t, myStreamB.Size() == MEGA_SIZE)

	bufferB, err := myStreamB.GetData()
	assert.NoError(t, err)
	cfb.Save(filename2)
	cfb.Close()

	cfc, err := mcdf.Open(filename2)
	assert.NoError(t, err)

	myStreamC, err := cfc.RootStorage().GetStream("MySecondStream")
	assert.NoError(t, err)

	assert.True(t, myStreamC.Size() == MEGA_SIZE, "DATA SIZE FAILED")

	bufferC, err := myStreamC.GetData()
	assert.NoError(t, err)
	assert.Equal(t, bufferB, bufferC, "DATA INTEGRITY FAILED")

	cfc.Close()

	if _, err = os.Stat(filename1); err == nil {
		err = os.Remove(filename1)
		assert.NoError(t, err)
	}
	if _, err = os.Stat(filename2); err == nil {
		err = os.Remove(filename2)
		assert.NoError(t, err)
	}
}

func Test_RE_WRITE_SMALLER_STREAM(t *testing.T) {
	const BUFFER_LENGTH = 8000

	const filename1 = "files/report.xls"
	const filename2 = "files/reportRW_SMALL.xls"

	b := GenBuffer(BUFFER_LENGTH)

	cf, err := mcdf.Open(filename1)
	assert.NoError(t, err)
	foundStream, err := cf.RootStorage().GetStream("Workbook")
	assert.NoError(t, err)
	err = foundStream.SetData(b)
	assert.NoError(t, err)
	err = cf.Save(filename2)
	assert.NoError(t, err)
	cf.Close()

	cf, err = mcdf.Open(filename2)
	assert.NoError(t, err)
	foundStream, err = cf.RootStorage().GetStream("Workbook")
	assert.NoError(t, err)
	c, err := foundStream.GetData()
	assert.NoError(t, err)
	assert.True(t, len(c) == BUFFER_LENGTH)
	cf.Close()

	if _, err = os.Stat(filename2); err == nil {
		err = os.Remove(filename2)
		assert.NoError(t, err)
	}
}

func Test_RE_WRITE_SMALLER_MINI_STREAM(t *testing.T) {
	mcdf.SetLogger(t)

	const filename1 = "files/report.xls"
	const filename2 = "files/RE_WRITE_SMALLER_MINI_STREAM.xls"

	var TEST_LENGTH int
	var b []byte
	{
		cf, err := mcdf.Open(filename1)
		assert.NoError(t, err)
		foundStream, err := cf.RootStorage().GetStream("\x05SummaryInformation")
		assert.NoError(t, err)
		TEST_LENGTH = int(foundStream.Size()) - 20
		b = GenBuffer(TEST_LENGTH)
		err = foundStream.SetData(b)
		assert.NoError(t, err)

		err = cf.Save(filename2)
		assert.NoError(t, err)
		cf.Close()
	}
	{
		mcdf.SetLogger(t)
		cf, err := mcdf.Open(filename2)
		assert.NoError(t, err)
		foundStream, err := cf.RootStorage().GetStream("\x05SummaryInformation")
		assert.NoError(t, err)
		c, err := foundStream.GetData()
		assert.NoError(t, err)
		assert.True(t, len(c) == int(TEST_LENGTH))
		assert.Equal(t, c, b)
		cf.Close()
	}

	if _, err := os.Stat(filename2); err == nil {
		err = os.Remove(filename2)
		assert.NoError(t, err)
	}
}

func Test_TRANSACTED_ADD_STREAM_TO_EXISTING_FILE(t *testing.T) {
	mcdf.SetLogger(t)

	const srcFilename = "files/report.xls"
	const dstFilename = "files/reportOverwrite.xls"

	_, err := Copy(srcFilename, dstFilename)
	assert.NoError(t, err)

	cf, err := mcdf.Open(dstFilename)
	assert.NoError(t, err)

	buffer := GenBuffer(5000)

	addedStream, err := cf.RootStorage().AddStream("MyNewStream")
	assert.NoError(t, err)

	err = addedStream.SetData(buffer)
	assert.NoError(t, err)

	err = cf.Commit()
	assert.NoError(t, err)

	cf.Close()

	if _, err := os.Stat(dstFilename); err == nil {
		err = os.Remove(dstFilename)
		assert.NoError(t, err)
	}
}

func Test_TRANSACTED_ADD_REMOVE_MULTIPLE_STREAM_TO_EXISTING_FILE(t *testing.T) {
	const srcFilename = "files/report.xls"
	const dstFilename = "files/reportOverwriteMultiple.xls"

	_, err := Copy(srcFilename, dstFilename)
	assert.NoError(t, err)

	cf, err := mcdf.Open(dstFilename)
	assert.NoError(t, err)

	buffer := GetBuffer(1995, 1)
	for i := 0; i < 254; i++ {
		addedStream, err := cf.RootStorage().AddStream(fmt.Sprintf("MyNewStream%v", i))
		assert.NoError(t, err)

		assert.NotNil(t, addedStream, "Stream not found")
		addedStream.SetData(buffer)

		b, err := addedStream.GetData()
		assert.NoError(t, err)
		assert.Equal(t, b, buffer, "Data buffer corrupted")
	}

	err = cf.Save(dstFilename + "PP")
	assert.NoError(t, err)
	cf.Close()

	if _, err := os.Stat(dstFilename); err == nil {
		err = os.Remove(dstFilename)
		assert.NoError(t, err)
	}

	if _, err := os.Stat(dstFilename + "PP"); err == nil {
		err = os.Remove(dstFilename + "PP")
		assert.NoError(t, err)
	}
}

func Test_TRANSACTED_ADD_MINISTREAM_TO_EXISTING_FILE(t *testing.T) {
	mcdf.SetLogger(t)

	const srcFilename = "files/report.xls"
	const dstFilename = "files/reportOverwriteMultiple.xls"

	_, err := Copy(srcFilename, dstFilename)
	assert.NoError(t, err)

	cf, err := mcdf.Open(dstFilename)
	assert.NoError(t, err)

	buffer := GetBuffer(31, 0x0A)

	myStream, err := cf.RootStorage().AddStream("MyStream")
	assert.NoError(t, err)

	err = myStream.SetData(buffer)
	assert.NoError(t, err)

	err = cf.Commit()
	assert.NoError(t, err)
	cf.Close()

	larger, err := ioutil.ReadFile(dstFilename)
	assert.NoError(t, err)
	smaller, err := ioutil.ReadFile(srcFilename)

	// Equal condition if minisector can be "allocated"
	// within the existing standard sector border
	assert.True(t, len(larger) >= len(smaller))

	if _, err := os.Stat(dstFilename); err == nil {
		err = os.Remove(dstFilename)
		assert.NoError(t, err)
	}

}

func Test_TRANSACTED_REMOVE_MINI_STREAM_ADD_MINISTREAM_TO_EXISTING_FILE(t *testing.T) {
	mcdf.SetLogger(t)

	const srcFilename = "files/report.xls"
	const dstFilename = "files/reportOverwrite2.xls"

	_, err := Copy(srcFilename, dstFilename)
	assert.NoError(t, err)

	cf, err := mcdf.Open(dstFilename)
	assert.NoError(t, err)

	err = cf.RootStorage().Delete("\x05SummaryInformation")
	assert.NoError(t, err)

	buffer := GenBuffer(2000)

	addedStream, err := cf.RootStorage().AddStream("MyNewStream")
	assert.NoError(t, err)
	err = addedStream.SetData(buffer)
	assert.NoError(t, err)

	err = cf.Commit()
	assert.NoError(t, err)
	cf.Close()

	if _, err := os.Stat(dstFilename); err == nil {
		err = os.Remove(dstFilename)
		assert.NoError(t, err)
	}

}

func Test_DELETE_STREAM_1(t *testing.T) {
	mcdf.SetLogger(t)

	const filename = "files/MultipleStorage.cfs"
	const dstFilename = "files/MultipleStorage_REMOVED_STREAM_1.cfs"

	cf, err := mcdf.Open(filename)
	assert.NoError(t, err)

	cfs, err := cf.RootStorage().GetStorage("MyStorage")
	assert.NoError(t, err)
	err = cfs.Delete("MySecondStream")
	assert.Error(t, err)
	assert.Equal(t, err, mcdf.NotFoundDirectory)

	err = cf.Save(dstFilename)
	assert.NoError(t, err)

	cf.Close()
	if _, err := os.Stat(dstFilename); err == nil {
		err = os.Remove(dstFilename)
		assert.NoError(t, err)
	}
}

func Test_DELETE_STREAM_2(t *testing.T) {
	mcdf.SetLogger(t)

	const filename = "files/MultipleStorage.cfs"
	const dstFilename = "files/MultipleStorage_REMOVED_STREAM_2.cfs"

	cf, err := mcdf.Open(filename)
	assert.NoError(t, err)
	cfs, err := cf.RootStorage().GetStorage("MyStorage")
	assert.NoError(t, err)
	cfs, err = cfs.GetStorage("AnotherStorage")
	assert.NoError(t, err)

	cfs.Delete("AnotherStream")
	assert.NoError(t, err)

	err = cf.Save(dstFilename)
	assert.NoError(t, err)

	cf.Close()
	if _, err := os.Stat(dstFilename); err == nil {
		err = os.Remove(dstFilename)
		assert.NoError(t, err)
	}
}

func Test_WRITE_AND_READ_CFS(t *testing.T) {
	mcdf.SetLogger(t)

	const filename = "files/WRITE_AND_READ_CFS.cfs"

	cf, err := mcdf.New(3)
	assert.NoError(t, err)

	st, err := cf.RootStorage().AddStorage("MyStorage")
	assert.NoError(t, err)
	sm, err := st.AddStream("MyStream")
	assert.NoError(t, err)
	b := GetBuffer(220, 0x0A)
	sm.SetData(b)

	err = cf.Save(filename)
	assert.NoError(t, err)
	cf.Close()

	cf2, err := mcdf.Open(filename)
	assert.NoError(t, err)
	st2, err := cf2.RootStorage().GetStorage("MyStorage")
	assert.NoError(t, err)
	sm2, err := st2.GetStream("MyStream")
	assert.NoError(t, err)
	assert.NotNil(t, sm2)
	assert.True(t, sm2.Size() == 220)

	b2, err := sm2.GetData()
	assert.NoError(t, err)
	assert.Equal(t, b, b2)

	cf2.Close()

	if _, err := os.Stat(filename); err == nil {
		err = os.Remove(filename)
		assert.NoError(t, err)
	}
}

func Test_INCREMENTAL_SIZE_MULTIPLE_WRITE_AND_READ_CFS(t *testing.T) {

	for i := random(1, 100); i < 1024*1024*70; i = i << 1 {
		SingleWriteReadMatching(t, i+random(0, 3))
	}
}

func SingleWriteReadMatching(t *testing.T, size int) {

	const filename = "files/INCREMENTAL_SIZE_MULTIPLE_WRITE_AND_READ_CFS.cfs"

	cf, err := mcdf.New(3)
	assert.NoError(t, err)
	st, err := cf.RootStorage().AddStorage("MyStorage")
	assert.NoError(t, err)
	sm, err := st.AddStream("MyStream")
	assert.NoError(t, err)

	b := GenBuffer(size)

	sm.SetData(b)
	cf.Save(filename)
	cf.Close()
	cf2, err := mcdf.Open(filename)
	assert.NoError(t, err)
	st2, err := cf2.RootStorage().GetStorage("MyStorage")
	assert.NoError(t, err)
	sm2, err := st2.GetStream("MyStream")
	assert.NoError(t, err)

	assert.NotNil(t, sm2)
	assert.True(t, sm2.Size() == int64(size))

	b2, err := sm2.GetData()
	assert.NoError(t, err)

	assert.Equal(t, b2, b)

	cf2.Close()

	if _, err := os.Stat(filename); err == nil {
		err = os.Remove(filename)
		assert.NoError(t, err)
	}
}

func Test_APPEND_DATA_TO_STREAM(t *testing.T) {
	const filename = "files/APPEND_DATA_TO_STREAM.cfs"

	b := []byte{0x0, 0x1, 0x2, 0x3}
	b2 := []byte{0x4, 0x5, 0x6, 0x7}

	cf, err := mcdf.New(3)
	assert.NoError(t, err)
	st, err := cf.RootStorage().AddStream("MyMiniStream")
	assert.NoError(t, err)
	err = st.SetData(b)
	assert.NoError(t, err)
	err = st.Append(b2)
	assert.NoError(t, err)

	cf.Save(filename)
	cf.Close()

	cmp := []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7}
	cf, err = mcdf.Open(filename)
	st, err = cf.RootStorage().GetStream("MyMiniStream")

	data, err := st.GetData()

	assert.Equal(t, cmp, data)
	cf.Close()

	if _, err := os.Stat(filename); err == nil {
		err = os.Remove(filename)
		assert.NoError(t, err)
	}
}
