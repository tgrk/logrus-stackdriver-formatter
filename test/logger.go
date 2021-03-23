package test

import "github.com/sirupsen/logrus"

// LogWrapper is for testing StackSkip. See stackskip_test.go for details
type LogWrapper struct {
	Logger *logrus.Logger
}

// Error is for testing StackSkip. See stackskip_test.go for details
func (l *LogWrapper) Error(msg string) {
	l.Logger.WithField("trace", "105445aa7843bc8bf206b12000100000/1;o=1").Error(msg)
}
