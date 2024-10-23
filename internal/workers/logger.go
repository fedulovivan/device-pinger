package workers

import (
	"log/slog"
	"net"
	"strings"

	"github.com/fedulovivan/mhz19-go/pkg/utils"
)

type WorkerLogger struct {
	Logger *slog.Logger
	tag    utils.Tag
}

func (l WorkerLogger) Fatalf(format string, v ...interface{}) {

	// before optimization:
	//   var err = fmt.Sprintf("%v", v)
	//   if strings.Contains(err, "host is down") || strings.Contains(err, "no route to host") {
	//   	return
	//   }
	// after:
	//   1. do not coerse entire v to string for checking text, pass only for *net.OpError
	//   2. pick directly underlying error from net.OpError.Err and then use more effective .HasPrefix instead of .Contains
	if err, ok := v[0].(*net.OpError); ok {
		errorString := err.Err.Error()
		shouldSkip := strings.HasPrefix(errorString, "sendto: host is down") || strings.HasPrefix(errorString, "sendto: no route to host")
		// fmt.Printf("%T [%v] shouldSkip=%v\n", err, errorString, shouldSkip)
		if shouldSkip {
			return
		}
	}

	l.Logger.Error(l.tag.F(format, v...))
}

func (l WorkerLogger) Errorf(format string, v ...interface{}) {
	l.Logger.Error(l.tag.F(format, v...))
}

func (l WorkerLogger) Warnf(format string, v ...interface{}) {
	l.Logger.Warn(l.tag.F(format, v...))
}

func (l WorkerLogger) Infof(format string, v ...interface{}) {
	l.Logger.Info(l.tag.F(format, v...))
}

func (l WorkerLogger) Debugf(format string, v ...interface{}) {
	l.Logger.Debug(l.tag.F(format, v...))
}
