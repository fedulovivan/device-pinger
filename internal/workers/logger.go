package workers

import (
	"log/slog"
	"net"
	"strings"

	"github.com/fedulovivan/mhz19-go/pkg/utils"
)

type SlogAdapter struct {
	tag utils.Tag
}

func (l SlogAdapter) Fatalf(format string, v ...interface{}) {

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

	slog.Error(l.tag.F(format, v...))
}

func (l SlogAdapter) Errorf(format string, v ...interface{}) {
	slog.Error(l.tag.F(format, v...))
}

func (l SlogAdapter) Warnf(format string, v ...interface{}) {
	slog.Warn(l.tag.F(format, v...))
}

func (l SlogAdapter) Infof(format string, v ...interface{}) {
	slog.Info(l.tag.F(format, v...))
}

func (l SlogAdapter) Debugf(format string, v ...interface{}) {
	slog.Debug(l.tag.F(format, v...))
}
