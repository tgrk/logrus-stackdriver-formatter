// Package stackdriver provides an adapter to the
// go-kit log.Logger interface.
package stackdriver

import (
	"fmt"
	"io"

	gokitlog "github.com/go-kit/kit/log"
	"github.com/sirupsen/logrus"
)

// LogrusGoKitLogger is a gokit-compatible wrapper for logrus.LogrusGoKitLogger
type LogrusGoKitLogger struct {
	Logger *logrus.Logger
}

// NewStackdriverLogger creates a gokit-compatible logger
func NewLogrusGoKitLogger(w io.Writer, opts ...Option) *LogrusGoKitLogger {
	logger := logrus.New()
	logger.SetFormatter(NewFormatter(opts...))
	logger.SetOutput(w)
	return &LogrusGoKitLogger{Logger: logger}
}

const msgKey = "msg"
const messageKey = "message"
const errKey = "err"
const errorKey = "error"
const severityKey = "severity"
const levelKey = "level"

// NewEntry creates a new logrus entry
func (l LogrusGoKitLogger) NewEntry(kvs ...interface{}) *logrus.Entry {
	return logrus.NewEntry(l.Logger)
}

// Log creates a log event from keyvals, a variadic sequence of alternating
// keys and values. It implements the fundamental go-kit Logger interface
func (l LogrusGoKitLogger) Log(keyvals ...interface{}) error {
	entry := l.NewEntry()
	fields, level, msg := extractLogElements(keyvals...)

	entry = entry.WithFields(fields)
	entry.Log(level, msg)

	return nil
}

// extractLogElements iterates through the keyvals to form well
// structuredkey:value pairs that Logrus expects. It also checks for keys with
// special meaning like "msg" and "level" to format the log entry
func extractLogElements(keyVals ...interface{}) (fields logrus.Fields, level logrus.Level, msg string) {
	msg = ""
	fields = logrus.Fields{}
	level = logrus.DebugLevel

	for i := 0; i < len(keyVals); i += 2 {
		fieldKey := fmt.Sprint(keyVals[i])
		if i+1 < len(keyVals) {

			fieldValue := fmt.Sprint(keyVals[i+1])
			if (fieldKey == msgKey || fieldKey == messageKey) && msg == "" {
				// if this is a "msg" key, store it separately so we can use it as the
				// main log message
				msg = fieldValue
			} else if (fieldKey == errKey || fieldKey == errorKey) {
				// if this is a "err" key, we should use the error message as
				// the main message and promote the level to Error
				err := fieldValue
				if err != "" {
					msg = err
					level = logrus.ErrorLevel
				}
			} else if fieldKey == levelKey || fieldKey == severityKey {
				// if this is a "level" key, it means GoKit logger is giving us
				// a hint to the logging level
				levelStr := fieldValue
				parsedLevel, err := logrus.ParseLevel(levelStr)
				if err != nil || level < parsedLevel {
					level = logrus.ErrorLevel
					fields[levelKey] = levelStr
				} else {
					level = parsedLevel
				}
			} else {
				// this is just regular log data, add it as a key:value pair
				fields[fieldKey] = keyVals[i+1]
			}
		} else {
			// odd pair key, with no matching value
			fields[fieldKey] = gokitlog.ErrMissingValue
		}
	}
	return
}
