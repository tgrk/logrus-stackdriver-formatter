// Package ctxlogrus wraps the go-grpc-middleware ctxlogrus, extracts a trace
// context to correlate logs emitted to the correct trace and span
package ctxlogrus

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

var (
	AddFields = ctxlogrus.AddFields
	ToContext = ctxlogrus.ToContext
)

// Extract provides a request-scoped log entry with details of the current
// trace in place.
func Extract(ctx context.Context) *logrus.Entry {
	entry := ctxlogrus.Extract(ctx)

	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return entry
	}

	return entry.WithField("span_context", sc)
}
