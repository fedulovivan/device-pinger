package workers

import "log"

type WorkerLogger struct {
	Logger *log.Logger
	target string
}

func (l WorkerLogger) Fatalf(format string, v ...interface{}) {
	l.Logger.Printf("[WORKER:"+l.target+"] "+format, v...)
}

func (l WorkerLogger) Errorf(format string, v ...interface{}) {
	l.Logger.Printf("[WORKER:"+l.target+"] "+format, v...)
}

func (l WorkerLogger) Warnf(format string, v ...interface{}) {
	l.Logger.Printf("[WORKER:"+l.target+"] "+format, v...)
}

func (l WorkerLogger) Infof(format string, v ...interface{}) {
	l.Logger.Printf("[WORKER:"+l.target+"] "+format, v...)
}

func (l WorkerLogger) Debugf(format string, v ...interface{}) {
	l.Logger.Printf("[WORKER:"+l.target+"] "+format, v...)
}
