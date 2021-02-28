package logadapter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/Mattel/logrus-stackdriver-formatter/test"
	"github.com/google/go-cmp/cmp"
	"github.com/sirupsen/logrus"
)

func TestStackSkip(t *testing.T) {
	var out bytes.Buffer

	logger := logrus.New()
	logger.Out = &out
	logger.Formatter = NewFormatter(
		WithProjectID("test-project"),
		WithService("test"),
		WithVersion("0.1"),
		WithStackSkip("github.com/Mattel/logrus-stackdriver-formatter"),
		WithSkipTimestamp(),
	)

	mylog := test.LogWrapper{
		Logger: logger,
	}
	mylog.Error("my log entry")

	want := map[string]interface{}{
		"@type":                               reportedErrorEventType,
		"severity":                            "ERROR",
		"message":                             "my log entry",
		"logName":                             "projects/test-project/logs/test",
		"logging.googleapis.com/trace":        "projects/test-project/traces/105445aa7843bc8bf206b12000100000",
		"logging.googleapis.com/spanId":       "1",
		"logging.googleapis.com/traceSampled": true,
		"serviceContext": map[string]interface{}{
			"service": "test",
			"version": "0.1",
		},
		"context": map[string]interface{}{
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
	}
	var got map[string]interface{}
	err := json.Unmarshal(out.Bytes(), &got)
	if err != nil {
		t.Error(err)
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Unexpected output (-want +got):\n%s", diff)
	}
}
