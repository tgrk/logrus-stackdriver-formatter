package logadapter

import (
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

var _ logrus.Hook = (*SpanHook)(nil)

type SpanHook struct{}

func (s *SpanHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (s *SpanHook) Fire(e *logrus.Entry) error {
	e.Data["span_context"] = trace.SpanContextFromContext(e.Context)

	return nil
}
