package logadapterexample

import (
	"os"
	"strconv"

	stackdriver "github.com/Mattel/logrus-stackdriver-formatter"
	"github.com/sirupsen/logrus"
)

func ExampleWithError() {
	logger := logrus.New()
	logger.Out = os.Stdout
	logger.Formatter = stackdriver.NewFormatter(
		stackdriver.WithService("test-service"),
		stackdriver.WithVersion("v0.1.0"),
		stackdriver.WithSkipTimestamp(),
	)

	logger.Info("application up and running")

	_, err := strconv.ParseInt("text", 10, 64)
	if err != nil {
		logger.WithError(err).WithField("trace", "1").Errorln("unable to parse integer")
	}

	// Output:
	// {"message":"application up and running","severity":"INFO","context":{}}
	// {"serviceContext":{"service":"test-service","version":"v0.1.0"},"message":
	// "unable to parse integer\nstrconv.ParseInt:
	// parsing \"text\": invalid syntax","severity":"ERROR","context":
	// {"reportLocation":
	// {"file":"testing/example_test.go","line":121,"function":"runExample"}},
	// "sourceLocation":
	// {"file":"testing/example_test.go","line":121,"function":"runExample"}}
}
