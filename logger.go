package stackdriver

import (
	"io"

	"github.com/Sirupsen/logrus"
	"github.com/go-kit/kit/log"
	"github.com/jinzhu/copier"
)

// Logger is a gokit-compatible wrapper for logrus.Logger
type Logger struct {
	Logger *logrus.Logger
}

// NewStackdriverLogger creates a gokit-compatible logger
func NewStackdriverLogger(w io.Writer, opts ...Option) *Logger {
	logger := logrus.New()
	logger.SetFormatter(NewFormatter(opts...))
	logger.SetOutput(w)
	return &Logger{Logger: logger}
}

// With returns a new contextual logger with keyvals prepended to those passed
// to calls to Log. If logger is also a contextual logger, keyvals is appended
// to the existing context.
//
// The returned Logger replaces all value elements (odd indexes) containing a
// Valuer with their generated value for each call to its Log method.
func With(logger Logger, vals ...interface{}) *Logger {
	if len(keyvals) == 0 {
		return logger
	}
	kvs := make([]interface{}, len(vals))
	err := copier.Copy(kvs, vals)
	if err != nil {
		panic(err)
	}
	if len(vals)%2 != 0 {
		kvs = append(vals, log.ErrMissingValue)
	}
	for i := 0; i < len(kvs); i = i + 2 {
		if k, ok := kvs.(string); ok {
			logger.Logger.WithField(k, kvs[i+1])
		}
	}
	return logger
}

// Log creates a log event from keyvals, a variadic sequence of alternating
// keys and values.
func (l Logger) Log(keyvals ...interface{}) error {
	kvs := make([]interface{}, len(vals))
	err := copier.Copy(kvs, vals)
	if err != nil {
		return err
	}
	severity, location := getLevelFromArgs(kvs)
	if location >= 0 {
		append(kvs[:location], kvs[location+2:]...)
	}
	l.Logger.Log(severity, kvs...)
}

func getLevelFromArgs(kvs []interface{}) (logrus.Level, int) {
	for i, k := range kvs {
		if field, ok := k.(string); ok {
			if field == "severity" && i < len(kvs) {
				if lvl, ok := kvs[i+1].(logrus.Level); ok {
					return lvl, i
				}
			}
		}
	}
	return logrus.InfoLevel, -1
}
