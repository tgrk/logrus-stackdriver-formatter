package stackdriver

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-stack/stack"
	"github.com/sirupsen/logrus"
)

type severity string

const (
	severityDebug    severity = "DEBUG"
	severityInfo     severity = "INFO"
	severityWarning  severity = "WARNING"
	severityError    severity = "ERROR"
	severityCritical severity = "CRITICAL"
	severityAlert    severity = "ALERT"
)

var levelsToSeverity = map[logrus.Level]severity{
	logrus.DebugLevel: severityDebug,
	logrus.InfoLevel:  severityInfo,
	logrus.WarnLevel:  severityWarning,
	logrus.ErrorLevel: severityError,
	logrus.FatalLevel: severityCritical,
	logrus.PanicLevel: severityAlert,
}

type serviceContext struct {
	Service string `json:"service,omitempty"`
	Version string `json:"version,omitempty"`
}

type ReportLocation struct {
	FilePath     string `json:"filePath,omitempty"`
	LineNumber   int    `json:"lineNumber,omitempty"`
	FunctionName string `json:"functionName,omitempty"`
}

type context struct {
	Data           map[string]interface{} `json:"data,omitempty"`
	ReportLocation *ReportLocation        `json:"reportLocation,omitempty"`
	HTTPRequest    map[string]interface{} `json:"httpRequest,omitempty"`
	PubSubRequest  map[string]interface{} `json:"pubSubRequest,omitempty"`
	GRPCRequest    map[string]interface{} `json:"grpcRequest,omitempty"`
}

type entry struct {
	Timestamp      string          `json:"timestamp,omitempty"`
	ServiceContext *serviceContext `json:"serviceContext,omitempty"`
	Message        string          `json:"message,omitempty"`
	Severity       severity        `json:"severity,omitempty"`
	Context        *context        `json:"context,omitempty"`
}

// Formatter implements Stackdriver formatting for logrus.
type Formatter struct {
	Service       string
	Version       string
	StackSkip     []string
	SkipTimestamp bool
	RegexSkip     string
}

// Option lets you configure the Formatter.
type Option func(*Formatter)

// WithService lets you configure the service name used for error reporting.
func WithService(n string) Option {
	return func(f *Formatter) {
		f.Service = n
	}
}

// WithVersion lets you configure the service version used for error reporting.
func WithVersion(v string) Option {
	return func(f *Formatter) {
		f.Version = v
	}
}

// WithStackSkip lets you configure which packages should be skipped for locating the error.
func WithStackSkip(v string) Option {
	return func(f *Formatter) {
		f.StackSkip = append(f.StackSkip, v)
	}
}

// WithRegexSkip lets you configure which functions or packages should be skipped for locating the error.
func WithRegexSkip(v string) Option {
	return func(f *Formatter) {
		f.RegexSkip = v
	}
}

// WithSkipTimestamp lets you avoid setting the timestamp
func WithSkipTimestamp() Option {
	return func(f *Formatter) {
		f.SkipTimestamp = true
	}
}

// NewFormatter returns a new Formatter.
func NewFormatter(options ...Option) *Formatter {
	fmtr := Formatter{
		StackSkip: []string{
			"github.com/sirupsen/logrus",
			"github.com/pbabbicola/logrus-stackdriver-formatter",
		},
	}
	for _, option := range options {
		option(&fmtr)
	}
	return &fmtr
}

func (f *Formatter) errorOrigin() (stack.Call, error) {
	skip := func(pkg string) bool {
		for _, skip := range f.StackSkip {
			if pkg == skip {
				return true
			}
		}
		return false
	}

	var r *regexp.Regexp
	if len(f.RegexSkip) != 0 {
		r = regexp.MustCompile(f.RegexSkip)
	}

	for i := 0; ; i++ {
		c := stack.Caller(i)
		// ErrNoFunc indicates we're over traversing the stack.
		if _, err := c.MarshalText(); err != nil {
			return stack.Call{}, nil
		}
		pkg := fmt.Sprintf("%+k", c)
		// Remove vendoring from package path.
		parts := strings.SplitN(pkg, "/vendor/", 2)
		pkg = parts[len(parts)-1]
		if !skip(pkg) && (r == nil || !r.MatchString(c.Frame().Function)) {
			return c, nil
		}
	}
}

// Format formats a logrus entry according to the Stackdriver specifications.
func (f *Formatter) Format(e *logrus.Entry) ([]byte, error) {
	severity := levelsToSeverity[e.Level]

	message := []string{}

	ee := entry{
		Severity: severity,
		Context: &context{
			Data: e.Data,
		},
	}
	if len(e.Message) > 0 {
		message = append(message, e.Message)
	}

	if !f.SkipTimestamp {
		if !e.Time.IsZero() {
			ee.Timestamp = e.Time.UTC().Format(time.RFC3339Nano)
		} else {
			ee.Timestamp = time.Now().UTC().Format(time.RFC3339Nano)
		}
	}

	switch severity {
	case severityError, severityCritical, severityAlert:
		ee.ServiceContext = &serviceContext{
			Service: f.Service,
			Version: f.Version,
		}

		// When using WithError(), the error is sent separately, but Error
		// Reporting expects it to be a part of the message so we append it
		// instead.
		if err, ok := ee.Context.Data["error"]; ok {
			message = append(message, fmt.Sprintf("%v", err))
			delete(ee.Context.Data, "error")
		}

		// If we supplied a stack trace, we can append it to the message
		if st, ok := ee.Context.Data["stackTrace"]; ok {
			message = append(message, fmt.Sprintf("%v", st))
			delete(ee.Context.Data, "stackTrace")
		}

		// As a convenience, when using supplying the httpRequest field, it
		// gets special care.
		if reqData, ok := ee.Context.Data["httpRequest"]; ok {
			if req, ok := reqData.(map[string]interface{}); ok {
				ee.Context.HTTPRequest = req
				delete(ee.Context.Data, "httpRequest")
			}
		}

		// As a convenience, when using supplying the grpcRequest field, it
		// gets special care.
		if reqData, ok := ee.Context.Data["grpcRequest"]; ok {
			if req, ok := reqData.(map[string]interface{}); ok {
				ee.Context.GRPCRequest = req
				delete(ee.Context.Data, "grpcRequest")
			}
		}
		// As a convenience, when using supplying the pubSubRequest field, it
		// gets special care.
		if reqData, ok := ee.Context.Data["pubSubRequest"]; ok {
			if req, ok := reqData.(map[string]interface{}); ok {
				ee.Context.PubSubRequest = req
				delete(ee.Context.Data, "pubsubRequest")
			}
		}

		if e.Caller != nil {
			ee.Context.ReportLocation = &ReportLocation{
				FilePath:     e.Caller.File,
				FunctionName: e.Caller.Function,
				LineNumber:   e.Caller.Line,
			}
		} else {
			// Extract report location from call stack.
			if c, err := f.errorOrigin(); err == nil {
				lineNumber, _ := strconv.ParseInt(fmt.Sprintf("%d", c), 10, 64)

				ee.Context.ReportLocation = &ReportLocation{
					FilePath:     fmt.Sprintf("%+s", c),
					LineNumber:   int(lineNumber),
					FunctionName: fmt.Sprintf("%n", c),
				}
			}
		}
	}

	ee.Message = strings.Join(message, "\n")

	b, err := json.Marshal(ee)
	if err != nil {
		return nil, err
	}

	return append(b, '\n'), nil
}
