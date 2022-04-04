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
				logadapter.WithProjectID("test-project"),
				logadapter.WithService("test"),
				logadapter.WithVersion("0.1"),
				logadapter.WithSkipTimestamp(),
				logadapter.WithSourceReference(
					"https://github.com/StevenACoffman/test.git",
					"v1.2.3",
				),
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
				WithField("trace", "105445aa7843bc8bf206b12000100000/1;o=1").
				Info("my log entry")
		},
		out: map[string]interface{}{
			"severity": "INFO",
			"message":  "my log entry",
			"logName":  "projects/test-project/logs/test",

			"logging.googleapis.com/trace":        "projects/test-project/traces/105445aa7843bc8bf206b12000100000",
			"logging.googleapis.com/spanId":       "1",
			"logging.googleapis.com/traceSampled": true,
			"context": map[string]interface{}{
				"data": map[string]interface{}{
					"foo": "bar",
				},
			},
			"sourceLocation": map[string]interface{}{
				"file":     "testing/testing.go",
				"function": "tRunner",
				"line":     1194.0,
			},
		},
	},
	{
		name: "WithField and Error",
		run: func(logger *logrus.Logger) {
			logger.
				WithField("foo", "bar").
				WithField("trace", "105445aa7843bc8bf206b12000100000/1;o=1").
				Error("my log entry")
		},
		out: map[string]interface{}{
			"@type":    "type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent",
			"severity": "ERROR",
			"message":  "my log entry",
			"logName":  "projects/test-project/logs/test",

			"logging.googleapis.com/trace":        "projects/test-project/traces/105445aa7843bc8bf206b12000100000",
			"logging.googleapis.com/spanId":       "1",
			"logging.googleapis.com/traceSampled": true,
			"serviceContext": map[string]interface{}{
				"service": "test",
				"version": "0.1",
			},
			"context": map[string]interface{}{
				"data": map[string]interface{}{
					"foo": "bar",
				},
				"sourceReferences": []map[string]interface{}{
					{
						"repository": "https://github.com/StevenACoffman/test.git",
						"revisionId": "v1.2.3",
					},
				},
				"reportLocation": map[string]interface{}{
					"filePath":     "testing/testing.go",
					"lineNumber":   1194.0,
					"functionName": "tRunner",
				},
			},
			"sourceLocation": map[string]interface{}{
				"file":     "testing/testing.go",
				"line":     1194.0,
				"function": "tRunner",
			},
		},
	},
	{
		name: "WithField, WithError and Error",
		run: func(logger *logrus.Logger) {
			logger.
				WithField("foo", "bar").
				WithField("trace", "105445aa7843bc8bf206b12000100000/1;o=1").
				WithError(errors.New("test error")).
				Error("my log entry")
		},
		out: map[string]interface{}{
			"@type":    "type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent",
			"severity": "ERROR",
			"message":  "my log entry\ntest error",
			"logName":  "projects/test-project/logs/test",

			"logging.googleapis.com/trace":        "projects/test-project/traces/105445aa7843bc8bf206b12000100000",
			"logging.googleapis.com/spanId":       "1",
			"logging.googleapis.com/traceSampled": true,
			"serviceContext": map[string]interface{}{
				"service": "test",
				"version": "0.1",
			},
			"context": map[string]interface{}{
				"data": map[string]interface{}{
					"foo":   "bar",
					"error": "test error",
				},
				"sourceReferences": []map[string]interface{}{
					{
						"repository": "https://github.com/StevenACoffman/test.git",
						"revisionId": "v1.2.3",
					},
				},
				"reportLocation": map[string]interface{}{
					"filePath":     "testing/testing.go",
					"lineNumber":   1194.0,
					"functionName": "tRunner",
				},
			},
			"sourceLocation": map[string]interface{}{
				"file":     "testing/testing.go",
				"line":     1194.0,
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
					"trace": "105445aa7843bc8bf206b12000100000/1;o=1",
					"httpRequest": map[string]interface{}{
						"requestMethod": "GET",
					},
				}).
				Error("my log entry")
		},
		out: map[string]interface{}{
			"@type":    "type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent",
			"severity": "ERROR",
			"message":  "my log entry",
			"logName":  "projects/test-project/logs/test",

			"logging.googleapis.com/trace":        "projects/test-project/traces/105445aa7843bc8bf206b12000100000",
			"logging.googleapis.com/spanId":       "1",
			"logging.googleapis.com/traceSampled": true,
			"serviceContext": map[string]interface{}{
				"service": "test",
				"version": "0.1",
			},
			"context": map[string]interface{}{
				"data": map[string]interface{}{
					"foo": "bar",
					"httpRequest": map[string]interface{}{
						"requestMethod": "GET",
					},
				},
				"reportLocation": map[string]interface{}{
					"filePath":     "testing/testing.go",
					"lineNumber":   1194.0,
					"functionName": "tRunner",
				},
				"sourceReferences": []map[string]interface{}{
					{
						"repository": "https://github.com/StevenACoffman/test.git",
						"revisionId": "v1.2.3",
					},
				},
			},
			"sourceLocation": map[string]interface{}{
				"file":     "testing/testing.go",
				"line":     1194.0,
				"function": "tRunner",
			},
		},
	},
}
