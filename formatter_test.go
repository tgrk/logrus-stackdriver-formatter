package stackdriver

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/sirupsen/logrus"
)

func TestFormatter(t *testing.T) {
	for _, tt := range formatterTests {
		var out bytes.Buffer

		logger := logrus.New()
		logger.Out = &out
		logger.Formatter = NewFormatter(
			WithService("test"),
			WithVersion("0.1"),
			WithSkipTimestamp(),
		)

		tt.run(logger)

		var got map[string]interface{}
		json.Unmarshal(out.Bytes(), &got)

		if !cmp.Equal(got, tt.out) {
			t.Errorf(cmp.Diff(got, tt.out))
		}
	}
}

var formatterTests = []struct {
	run func(*logrus.Logger)
	out map[string]interface{}
}{
	{
		run: func(logger *logrus.Logger) {
			logger.WithField("foo", "bar").Info("my log entry")
		},
		out: map[string]interface{}{
			"severity": "INFO",
			"message":  "my log entry",
			"context": map[string]interface{}{
				"data": map[string]interface{}{
					"foo": "bar",
				},
			},
		},
	},
	{
		run: func(logger *logrus.Logger) {
			logger.WithField("foo", "bar").Error("my log entry")
		},
		out: map[string]interface{}{
			"severity": "ERROR",
			"message":  "my log entry",
			"serviceContext": map[string]interface{}{
				"service": "test",
				"version": "0.1",
			},
			"context": map[string]interface{}{
				"data": map[string]interface{}{
					"foo": "bar",
				},
				"reportLocation": map[string]interface{}{
					"filePath":     "github.com/pbabbicola/logrus-stackdriver-formatter/formatter_test.go",
					"lineNumber":   57.0,
					"functionName": "glob..func2",
				},
			},
		},
	},
	{
		run: func(logger *logrus.Logger) {
			logger.
				WithField("foo", "bar").
				WithError(errors.New("test error")).
				Error("my log entry")
		},
		out: map[string]interface{}{
			"severity": "ERROR",
			"message":  "my log entry: test error",
			"serviceContext": map[string]interface{}{
				"service": "test",
				"version": "0.1",
			},
			"context": map[string]interface{}{
				"data": map[string]interface{}{
					"foo": "bar",
				},
				"reportLocation": map[string]interface{}{
					"filePath":     "github.com/pbabbicola/logrus-stackdriver-formatter/formatter_test.go",
					"lineNumber":   83.0,
					"functionName": "glob..func3",
				},
			},
		},
	},
	{
		run: func(logger *logrus.Logger) {
			logger.
				WithFields(logrus.Fields{
					"foo": "bar",
					"httpRequest": map[string]interface{}{
						"method": "GET",
					},
				}).
				Error("my log entry")
		},
		out: map[string]interface{}{
			"severity": "ERROR",
			"message":  "my log entry",
			"serviceContext": map[string]interface{}{
				"service": "test",
				"version": "0.1",
			},
			"context": map[string]interface{}{
				"data": map[string]interface{}{
					"foo": "bar",
				},
				"httpRequest": map[string]interface{}{
					"method": "GET",
				},
				"reportLocation": map[string]interface{}{
					"filePath":     "github.com/pbabbicola/logrus-stackdriver-formatter/formatter_test.go",
					"lineNumber":   113.0,
					"functionName": "glob..func4",
				},
			},
		},
	},
}
