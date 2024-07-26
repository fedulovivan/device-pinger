package workers

import (
	"fmt"
	"log/slog"
	"strings"
)

type WorkerLogger struct {
	Logger *slog.Logger
	target string
}

func (l WorkerLogger) Fatalf(format string, v ...interface{}) {
	var err = fmt.Sprintf("%v", v)
	if strings.Contains(err, "host is down") || strings.Contains(err, "no route to host") {
		return
	}
	l.Logger.Debug(fmt.Sprintf("[WORKER:"+l.target+"] "+format, v...))
}

func (l WorkerLogger) Errorf(format string, v ...interface{}) {
	l.Logger.Debug(fmt.Sprintf("[WORKER:"+l.target+"] "+format, v...))
}

func (l WorkerLogger) Warnf(format string, v ...interface{}) {
	l.Logger.Debug(fmt.Sprintf("[WORKER:"+l.target+"] "+format, v...))
}

func (l WorkerLogger) Infof(format string, v ...interface{}) {
	l.Logger.Debug(fmt.Sprintf("[WORKER:"+l.target+"] "+format, v...))
}

func (l WorkerLogger) Debugf(format string, v ...interface{}) {
	l.Logger.Debug(fmt.Sprintf("[WORKER:"+l.target+"] "+format, v...))
}
