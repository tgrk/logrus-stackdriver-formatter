package logadapter_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	logadapter "github.com/StevenACoffman/logrus-stackdriver-formatter"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestFormatter(t *testing.T) {
	for i := range formatterTests {
		tt := formatterTests[i]
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer

			logger := logrus.New()
			logger.Out = &out
			logger.Formatter = logadapter.NewFormatter(
				logadapter.WithService("test"),
				logadapter.WithVersion("0.1"),
				logadapter.WithSkipTimestamp(),
			)

			tt.run(logger)

			got, err := json.Marshal(tt.out)
			if err != nil {
				t.Error(err)
			}
			assert.JSONEq(t, string(got), out.String())
		})
	}
}

var formatterTests = []struct {
	run  func(*logrus.Logger)
	out  map[string]interface{}
	name string
}{
	{
		name: "With Field",
		run: func(logger *logrus.Logger) {
			logger.
				WithField("foo", "bar").
				WithField("trace", "1").
				Info("my log entry")
		},
		out: map[string]interface{}{
			"severity": "INFO",
			"message":  "my log entry",
			"logName":  "projects//logs/test",
			"trace":    "projects//traces/1",
			"context": map[string]interface{}{
				"data": map[string]interface{}{
					"foo":   "bar",
					"trace": "1",
				},
			},
		},
	},
	{
		name: "WithField and Error",
		run: func(logger *logrus.Logger) {
			logger.
				WithField("foo", "bar").
				WithField("trace", "1").
				Error("my log entry")
		},
		out: map[string]interface{}{
			"severity": "ERROR",
			"message":  "my log entry",
			"logName":  "projects//logs/test",
			"trace":    "projects//traces/1",
			"serviceContext": map[string]interface{}{
				"service": "test",
				"version": "0.1",
			},
			"context": map[string]interface{}{
				"data": map[string]interface{}{
					"foo":   "bar",
					"trace": "1",
				},
				"reportLocation": map[string]interface{}{
					"file":     "testing/testing.go",
					"line":     865.0,
					"function": "tRunner",
				},
			},
			"sourceLocation": map[string]interface{}{
				"file":     "testing/testing.go",
				"line":     865.0,
				"function": "tRunner",
			},
		},
	},
	{
		name: "WithField, WithError and Error",
		run: func(logger *logrus.Logger) {
			logger.
				WithField("foo", "bar").
				WithField("trace", "1").
				WithError(errors.New("test error")).
				Error("my log entry")
		},
		out: map[string]interface{}{
			"severity": "ERROR",
			"message":  "my log entry\ntest error",
			"logName":  "projects//logs/test",
			"trace":    "projects//traces/1",
			"serviceContext": map[string]interface{}{
				"service": "test",
				"version": "0.1",
			},
			"context": map[string]interface{}{
				"data": map[string]interface{}{
					"foo":   "bar",
					"trace": "1",
				},
				"reportLocation": map[string]interface{}{
					"file":     "testing/testing.go",
					"line":     865.0,
					"function": "tRunner",
				},
			},
			"sourceLocation": map[string]interface{}{
				"file":     "testing/testing.go",
				"line":     865.0,
				"function": "tRunner",
			},
		},
	},
	{
		name: "WithField, HTTPRequest and Error",
		run: func(logger *logrus.Logger) {
			logger.
				WithFields(logrus.Fields{
					"foo":   "bar",
					"trace": "1",
					"httpRequest": map[string]interface{}{
						"requestMethod": "GET",
					},
				}).
				Error("my log entry")
		},
		out: map[string]interface{}{
			"severity": "ERROR",
			"message":  "my log entry",
			"logName":  "projects//logs/test",
			"trace":    "projects//traces/1",
			"serviceContext": map[string]interface{}{
				"service": "test",
				"version": "0.1",
			},
			"context": map[string]interface{}{
				"data": map[string]interface{}{
					"foo":   "bar",
					"trace": "1",
					"httpRequest": map[string]interface{}{
						"requestMethod": "GET",
					},
				},
				"reportLocation": map[string]interface{}{
					"file":     "testing/testing.go",
					"line":     865.0,
					"function": "tRunner",
				},
			},
			"sourceLocation": map[string]interface{}{
				"file":     "testing/testing.go",
				"line":     865.0,
				"function": "tRunner",
			},
		},
	},
}
