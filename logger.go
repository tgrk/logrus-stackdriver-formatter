// Package stackdriver provides an adapter to the
// go-kit log.Logger interface.
package stackdriver

import (
	"fmt"

	gokitlog "github.com/go-kit/kit/log"
	"github.com/sirupsen/logrus"
)

// LogrusGoKitLogger is the concrete implementation of the logging wrapper
type LogrusGoKitLogger struct {
	logrusLogger
}

type logrusLogger interface {
	WithFields(fields logrus.Fields) *logrus.Entry
}

// NewLogrusGoKitLogger wraps a logrus instance and implements the GoKit Logger interface
func NewLogrusGoKitLogger(logger logrusLogger) *LogrusGoKitLogger {
	return &LogrusGoKitLogger{logger}
}



const msgKey = "msg"
const errKey = "err"
const levelKey = "level"

// Log implements the fundamental Logger interface
func (l LogrusGoKitLogger) Log(keyvals ...interface{}) error {
	fields, level, msg := l.extractLogElements(keyvals...)

	entry := l.WithFields(fields)
	entry.Log(level, msg)

	return nil
}

// extractLogElements iterates through the keyvals to form well
// structuredkey:value pairs that Logrus expects. It also checks for keys with
// special meaning like "msg" and "level" to format the log entry
func (l LogrusGoKitLogger) extractLogElements(keyvals ...interface{}) (fields logrus.Fields, level logrus.Level, msg string) {
	msg = ""
	fields = logrus.Fields{}
	level = logrus.DebugLevel

	for i := 0; i < len(keyvals); i += 2 {
		if i+1 < len(keyvals) {

			if fmt.Sprint(keyvals[i]) == msgKey && msg == "" {
				// if this is a "msg" key, store it separately so we can use it as the
				// main log message
				msg = fmt.Sprint(keyvals[i+1])
			} else if fmt.Sprint(keyvals[i]) == errKey {
				// if this is a "err" key, we should use the error message as
				// the main message and promote the level to Error
				err := fmt.Sprint(keyvals[i+1])
				if err != "" {
					msg = err
					level = logrus.ErrorLevel
				}
			} else if fmt.Sprint(keyvals[i]) == levelKey {
				// if this is a "level" key, it means GoKit logger is giving us
				// a hint to the logging level
				levelStr := fmt.Sprint(keyvals[i+1])
				parsedLevel, err := logrus.ParseLevel(levelStr)
				if err != nil || level < parsedLevel {
					level = logrus.ErrorLevel
					fields["level"] = levelStr
				} else {
					level = parsedLevel
				}
			} else {
				// this is just regular log data, add it as a key:value pair
				fields[fmt.Sprint(keyvals[i])] = keyvals[i+1]
			}
		} else {
			// odd pair key, with no matching value
			fields[fmt.Sprint(keyvals[i])] = gokitlog.ErrMissingValue
		}
	}
	return
}