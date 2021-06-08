package log

import (
	"io/ioutil"
	"log"
	"os"
	"sync"
)

// log.Lshortfile 支持显示文件名和行号
var (
	errLog = log.New(os.Stdout,"\033[31m[error]]\033[0m", log.LstdFlags|log.Lshortfile)
	infoLog = log.New(os.Stdout,"\033[34m[info]\033[0m", log.LstdFlags|log.Lshortfile)
	loggers = []*log.Logger{errLog, infoLog}

	mu sync.Mutex
)

var (
	Error = errLog.Println
	Errorf = errLog.Printf
	Info = infoLog.Println
	Infof = infoLog.Printf
)

// log levels
const (
	InfoLevel = iota
	ErrorLevel
	Distabled
)

func setLevel(level int) {
	mu.Lock()
	defer mu.Unlock()

	for _,logger := range loggers {
		logger.SetOutput(os.Stdout)
	}

	if ErrorLevel < level {
		errLog.SetOutput(ioutil.Discard)
	}

	if InfoLevel < level {
		infoLog.SetOutput(ioutil.Discard)
	}

}


