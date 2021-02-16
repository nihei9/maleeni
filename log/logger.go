package log

import (
	"fmt"
	"io"
)

type Logger interface {
	Log(format string, a ...interface{})
}

var (
	_ Logger = &logger{}
	_ Logger = &nopLogger{}
)

type logger struct {
	w io.Writer
}

func NewLogger(w io.Writer) (*logger, error) {
	if w == nil {
		return nil, fmt.Errorf("w is nil; NewLogger() needs a writer")
	}
	return &logger{
		w: w,
	}, nil
}

func (l *logger) Log(format string, a ...interface{}) {
	fmt.Fprintf(l.w, format+"\n", a...)
}

type nopLogger struct {
}

func NewNopLogger() *nopLogger {
	return &nopLogger{}
}

func (l *nopLogger) Log(format string, a ...interface{}) {
}
