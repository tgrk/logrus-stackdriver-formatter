// Package ctxlogrus wraps the go-grpc-middleware ctxlogrus, extracts a trace
// context to correlate logs emitted to the correct trace and span
package ctxlogrus

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/sirupsen/logrus"
)

var (
	AddFields = ctxlogrus.AddFields
	ToContext = ctxlogrus.ToContext
)

// Extract provides a request-scoped log entry with details of the current
// trace in place.
func Extract(ctx context.Context) *logrus.Entry {
	return ctxlogrus.Extract(ctx).WithContext(ctx)
}
