package stackdriver

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

type mockLogrusLogger struct{}

func (l *mockLogrusLogger) GetLevel() logrus.Level {
	return logrus.DebugLevel
}

func (l *mockLogrusLogger) WithFields(fields logrus.Fields) *logrus.Entry {
	return &logrus.Entry{}
}

func TestLogrusGoKitLogger_extractLogElements_basic(t *testing.T) {

	mockLogrus := &mockLogrusLogger{}
	logger := &LogrusGoKitLogger{mockLogrus}

	fields, level, msg := logger.extractLogElements("msg", "testy mctestface", "level", "error", "foo", "bar", "number", 42, "flag", true)

	expectedFields := logrus.Fields{}
	expectedFields["foo"] = "bar"
	expectedFields["number"] = 42
	expectedFields["flag"] = true

	assert.Equal(t, expectedFields, fields)
	assert.Equal(t, logrus.ErrorLevel, level)
	assert.Equal(t, "testy mctestface", msg)
}

func TestLogrusGoKitLogger_extractLogElements_defaultLevel(t *testing.T) {

	mockLogrus := &mockLogrusLogger{}
	logger := &LogrusGoKitLogger{mockLogrus}

	fields, level, msg := logger.extractLogElements("msg", "testy mctestface")

	expectedFields := logrus.Fields{}

	assert.Equal(t, expectedFields, fields)
	assert.Equal(t, logrus.DebugLevel, level)
	assert.Equal(t, "testy mctestface", msg)
}

func TestLogrusGoKitLogger_extractLogElements_errorOverride(t *testing.T) {

	mockLogrus := &mockLogrusLogger{}
	logger := &LogrusGoKitLogger{mockLogrus}

	fields, level, msg := logger.extractLogElements("err", "test error", "msg", "some message", "level", "debug", "number", 42, "flag", true)

	expectedFields := logrus.Fields{}
	expectedFields["msg"] = "some message"
	expectedFields["level"] = "debug"
	expectedFields["number"] = 42
	expectedFields["flag"] = true

	assert.Equal(t, expectedFields, fields)
	assert.Equal(t, logrus.ErrorLevel, level)
	assert.Equal(t, "test error", msg)
}