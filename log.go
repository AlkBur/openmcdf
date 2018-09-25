package openmcdf

var log Logger

type Logger interface {
	Log(args ...interface{})
}

func SetLogger(l Logger) {
	log = l
}

func Log(args ...interface{}) {
	if log != nil {
		log.Log(args...)
	}
}
