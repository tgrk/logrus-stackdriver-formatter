package logadapter

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

// InitLogging initializes a logger to send things to stackdriver.
func InitLogging() *logrus.Logger {
	var log = logrus.New()
	log.Formatter = NewFormatter()
	log.Level = logrus.DebugLevel
	log.SetOutput(os.Stdout)

	log.Info("Logger successfully initialized!")

	return log
}

// InitLogrusGoKitLogger initializes a go kit logger to send things to stackdriver.
func InitLogrusGoKitLogger(w io.Writer, opts ...Option) *LogrusGoKitLogger {
	logger := logrus.New()
	logger.SetFormatter(NewFormatter(opts...))
	logger.SetOutput(w)
	return NewLogrusGoKitLogger(logger)
}
