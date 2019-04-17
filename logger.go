package stackdriver

import (
	"io"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/sirupsen/logrus"
)

// Logger is a gokit-compatible wrapper for logrus.Logger
type Logger struct {
	Logger *logrus.Entry
}

// NewStackdriverLogger creates a gokit-compatible logger
func NewStackdriverLogger(w io.Writer, opts ...Option) *Logger {
	logger := logrus.New()
	logger.SetFormatter(NewFormatter(opts...))
	logger.SetOutput(w)
	return &Logger{Logger: logrus.NewEntry(logger)}
}

// With returns a new contextual logger with keyvals prepended to those passed
// to calls to Log. If logger is also a contextual logger, keyvals is appended
// to the existing context.
//
// The returned Logger replaces all value elements (odd indexes) containing a
// Valuer with their generated value for each call to its Log method.
func With(logger *Logger, vals ...interface{}) *Logger {
	if len(vals) == 0 {
		return logger
	}
	logger.Logger = logger.Logger.WithFields(valsToFields(vals))
	return logger
}

// WithFields appends fields to the Logger to be used on the
// logrus logger when logging
func (l *Logger) WithFields(f logrus.Fields) *Logger {
	l.Logger = l.Logger.WithFields(f)
	return l
}

// Log creates a log event from keyvals, a variadic sequence of alternating
// keys and values.
func (l Logger) Log(kvs ...interface{}) error {
	severity, location := getLevelFromArgs(kvs...)
	if location >= 0 {
		kvs = append(kvs[:location], kvs[location+2:]...)
	}
	message, location := getMessageFromArgs(kvs...)
	if location >= 0 {

		kvs = append(kvs[:location], kvs[location+2:]...)
	}
	log := l.Logger.WithFields(valsToFields(kvs...))
	log.Log(severity, message)
	return nil
}

func getLevelFromArgs(kvs ...interface{}) (logrus.Level, int) {
	for i, k := range kvs {
		if field, ok := k.(string); ok {
			if strings.ToLower(field) == "severity" && i < len(kvs) {
				if lvl, ok := kvs[i+1].(logrus.Level); ok {
					return lvl, i
				}
				if v, ok := kvs[i+1].(string); ok {
					if lvl, err := logrus.ParseLevel(v); err == nil {
						return lvl, i
					}
				}
			}
		}
	}
	return logrus.InfoLevel, -1
}

func getMessageFromArgs(kvs ...interface{}) (string, int) {
	for i, k := range kvs {
		if field, ok := k.(string); ok {
			if field == "message" && i < len(kvs) {
				if msg, ok := kvs[i+1].(string); ok {
					return msg, i
				}
			}
		}
	}
	return "", -1
}

func valsToFields(vals ...interface{}) logrus.Fields {
	kvs := make([]interface{}, len(vals))
	copy(kvs, vals)
	if len(vals)%2 != 0 {
		kvs = append(kvs, log.ErrMissingValue)
	}
	fields := logrus.Fields{}
	for i := 0; i < len(kvs)-1; i = i + 2 {
		if k, ok := kvs[i].(string); ok {
			fields[k] = kvs[i+1]
		}
	}
	return fields
}
