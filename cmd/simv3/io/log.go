package io

import (
	"fmt"
	"io"
	"log"

	"github.com/smartcontractkit/libocr/commontypes"
)

type LogLevel int

const (
	Critical LogLevel = iota
	Error
	Warn
	Info
	Debug
	Trace
)

var _ commontypes.Logger = (*SimpleLogger)(nil)

type SimpleLogger struct {
	loggers []*log.Logger
}

func NewSimpleLogger(out io.Writer, lvl LogLevel) *SimpleLogger {
	sl := &SimpleLogger{
		loggers: make([]*log.Logger, lvl+1),
	}

	// flags := log.Ldate | log.Ltime | log.Lmicroseconds | log.Llongfile

	for i := 0; i <= int(lvl); i++ {
		var prefix string
		switch LogLevel(i) {
		case Critical:
			prefix = "[critical] "
		case Error:
			prefix = "[error] "
		case Warn:
			prefix = "[warn] "
		case Info:
			prefix = "[info] "
		case Debug:
			prefix = "[debug] "
		case Trace:
			prefix = "[trace] "
		}
		l := log.New(out, prefix, log.Ldate|log.Ltime|log.Lmicroseconds)
		sl.loggers[i] = l
	}

	return sl
}

func (l *SimpleLogger) log(lvl LogLevel, msg string, fields commontypes.LogFields) {
	if int(lvl) >= len(l.loggers) {
		return
	}
	var color string
	var reset string

	switch lvl {
	case Critical:
		color = string(colorRed)
		reset = string(colorReset)
	case Error:
		color = string(colorYellow)
		reset = string(colorReset)
	case Trace:
		color = string(colorCyan)
		reset = string(colorReset)
	default:
		color = ""
		reset = ""
	}
	l.loggers[lvl].Print(color, msg, fmt.Sprint(toKeysAndValues(fields)...), reset)
}

func (l *SimpleLogger) Critical(msg string, fields commontypes.LogFields) {
	l.log(Critical, msg, fields)
}

func (l *SimpleLogger) Error(msg string, fields commontypes.LogFields) {
	l.log(Error, msg, fields)
}

func (l *SimpleLogger) Warn(msg string, fields commontypes.LogFields) {
	l.log(Warn, msg, fields)
}

func (l *SimpleLogger) Info(msg string, fields commontypes.LogFields) {
	l.log(Info, msg, fields)
}

func (l *SimpleLogger) Debug(msg string, fields commontypes.LogFields) {
	l.log(Debug, msg, fields)
}

func (l *SimpleLogger) Trace(msg string, fields commontypes.LogFields) {
	l.log(Trace, msg, fields)
}

func toKeysAndValues(fields commontypes.LogFields) []interface{} {
	out := []interface{}{}
	for key, val := range fields {
		out = append(out, fmt.Sprintf(", %s: ", key), val)
	}
	return out
}

const (
	colorReset string = "\033[0m"
	colorRed   string = "\033[31m"
	//colorGreen  string = "\033[32m"
	colorYellow string = "\033[33m"
	//colorBlue   string = "\033[34m"
	//colorPurple string = "\033[35m"
	colorCyan string = "\033[36m"
	//colorWhite string = "\033[37m"
)

type WrappedLogger struct {
	logger *SimpleLogger
}

func (l *WrappedLogger) Write(p []byte) (n int, err error) {
	l.logger.Debug(string(p), nil)
	n = len(p)
	return
}
