package stackdriver

import (
	"bytes"
	"encoding/json"
	"testing"
	"github.com/google/go-cmp/cmp"

	"github.com/StevenACoffman/logrus-stackdriver-formatter/test"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestStackSkip(t *testing.T) {
	var out bytes.Buffer

	logger := logrus.New()
	logger.Out = &out
	logger.Formatter = NewFormatter(
		WithService("test"),
		WithVersion("0.1"),
		WithStackSkip("github.com/StevenACoffman/logrus-stackdriver-formatter/test"),
		WithSkipTimestamp(),
	)

	mylog := test.LogWrapper{
		Logger: logger,
	}

	mylog.Error("my log entry")

	want := map[string]interface{}{
		"severity": "ERROR",
		"message":  "my log entry",
		"serviceContext": map[string]interface{}{
			"service": "test",
			"version": "0.1",
		},
		"context": map[string]interface{}{
			"reportLocation": map[string]interface{}{
				"filePath":     "github.com/StevenACoffman/logrus-stackdriver-formatter/stackskip_test.go",
				"lineNumber":   30.0,
				"functionName": "TestStackSkip",
			},
		},
		"sourceLocation": map[string]interface{}{
			"file":     "testing/testing.go",
			"line":     909.0,
			"function": "tRunner",
		},
	}

	if !cmp.Equal(got, want) {
		cmp.Diff(got, want)
	}
	assert.JSONEq(t, out.String(), string(got))
}
