package Test

import (
	mcdf "github.com/AlkBur/openmcdf"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_READ_MAIN_STREAM(t *testing.T) {
	var find *mcdf.Stream
	var st *mcdf.Storage
	const filename = "files/1Cv7.MD"

	cf, err := mcdf.Open(filename)
	defer cf.Close()
	assert.NoError(t, err)

	st, err = cf.RootStorage().GetStorage("Metadata")
	assert.NoError(t, err)
	assert.NotNil(t, st)

	find, err = st.GetStream("Main MetaData Stream")
	assert.NoError(t, err)
	assert.NotNil(t, find)

	temp, err := find.GetData()
	assert.NoError(t, err)

	assert.NotNil(t, temp)
	assert.True(t, len(temp) > 0)
}
