package logadapter

import (
	"context"
	"net/http"
	"strings"
)

var defaultLogOptions = &middlewareOptions{
	filterRPC:  DefaultFilterRPC,
	filterHTTP: DefaultFilterHTTP,
}

type MiddlewareOption func(*middlewareOptions)

// Options
type middlewareOptions struct {
	filterRPC  FilterRPC
	filterHTTP FilterHTTP
}

func evaluateMiddlewareOptions(opts []MiddlewareOption) *middlewareOptions {
	optCopy := &middlewareOptions{}
	*optCopy = *defaultLogOptions
	for _, o := range opts {
		o(optCopy)
	}
	return optCopy
}

// WithRPCFilter provides a filter to the logging middleware that determines
// whether or not to log individual messages
func WithRPCFilter(f FilterRPC) MiddlewareOption {
	return func(o *middlewareOptions) {
		o.filterRPC = f
	}
}

// WithHTTPFilter provides a filter to the logging middleware that determines
// whether or not to log individual messages
func WithHTTPFilter(f FilterHTTP) MiddlewareOption {
	return func(o *middlewareOptions) {
		o.filterHTTP = f
	}
}

// Logging filters
type (
	FilterRPC  func(ctx context.Context, fullMethod string, err error) bool
	FilterHTTP func(r *http.Request) bool
)

// TODO: default filters (NO healthchecks!)

func DefaultFilterRPC(_ context.Context, fullMethod string, _ error) bool {
	switch {
	case strings.HasPrefix(fullMethod, "/grpc.health"):
		return false
	case strings.HasPrefix(fullMethod, "/grpc.reflection"):
		return false
	default:
		return true
	}
}

func DefaultFilterHTTP(r *http.Request) bool {
	switch {
	case strings.HasPrefix(r.URL.Path, "/health"):
		return false
	default:
		return true
	}
}
