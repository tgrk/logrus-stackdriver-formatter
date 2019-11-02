package logadapter

import (
	"io"

	"github.com/sirupsen/logrus"
)

// InitLogging initializes a logrus logger to send things to stackdriver.
func InitLogging(w io.Writer, opts ...Option) *logrus.Logger {
	log := logrus.New()
	log.Formatter = NewFormatter(opts...)
	log.Level = logrus.DebugLevel
	log.SetOutput(w)

	log.Info("Logger successfully initialized!")

	return log
}

// InitLogrusGoKitLogger initializes a go kit logger to send things to stackdriver.
func InitLogrusGoKitLogger(w io.Writer, opts ...Option) *LogrusGoKitLogger {
	logger := InitLogging(w, opts...)
	return NewLogrusGoKitLogger(logger)
}
