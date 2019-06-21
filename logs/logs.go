package logs

import (
	"io"
	"log"
)

var (
	Info  *log.Logger
	Debug *log.Logger
	Warn  *log.Logger
	Error *log.Logger
)

//InitLogger initializes 4 levels of logging and returns the callers back to the calling function
func InitLogger(infoHandle io.Writer, debugHandle io.Writer, warnHandle io.Writer, errorHandle io.Writer) (
	*log.Logger, *log.Logger, *log.Logger, *log.Logger) {
	Info = log.New(infoHandle, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	Debug = log.New(debugHandle, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	Warn = log.New(warnHandle, "WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(errorHandle, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	return Info, Debug, Warn, Error
}
