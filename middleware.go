package logadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/felixge/httpsnoop"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus/ctxlogrus"
	"github.com/lithammer/shortuuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

// LoggingMiddleware proivdes a request-scoped log entry into context for HTTP
// requests, writes request logs in a structured format to stackdriver.
func LoggingMiddleware(log *logrus.Logger, opts ...MiddlewareOption) func(http.Handler) http.Handler {
	o := evaluateMiddlewareOptions(opts)

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			entry := logrus.NewEntry(log)
			ctx := ctxlogrus.ToContext(r.Context(), entry)
			r = r.WithContext(ctx)

			// https://cloud.google.com/logging/docs/reference/v2/rest/v2/LogEntry#HttpRequest
			request := &HTTPRequest{
				RequestMethod: r.Method,
				RequestURL:    r.RequestURI,
				RemoteIP:      getRemoteIP(r),
				Referer:       r.Referer(),
				UserAgent:     r.UserAgent(),
				RequestSize:   strconv.FormatInt(r.ContentLength, 10),
				Protocol:      r.Proto,
			}
			ctxlogrus.AddFields(ctx, logrus.Fields{"httpRequest": request})

			traceHeader := r.Header.Get("X-Cloud-Trace-Context")
			if traceHeader != "" {
				ctxlogrus.AddFields(ctx, logrus.Fields{"trace": traceHeader})
			}

			m := httpsnoop.CaptureMetrics(handler, w, r)

			request.Status = strconv.Itoa(m.Code)
			request.Latency = fmt.Sprintf("%.9fs", m.Duration.Seconds())
			request.ResponseSize = strconv.FormatInt(m.Written, 10)

			if o.filterHTTP(r) {
				// log the result
				ctxlogrus.Extract(ctx).Infof("served HTTP %v %v", r.Method, r.URL)
			}
		})
	}
}

// UnaryLoggingInterceptor provides a request-scoped log entry into context for
// Unary gRPC requests, and logs request details on the response.
// Logging interceptors should be chained at the very top of the request scope.
func UnaryLoggingInterceptor(logger *logrus.Logger, opts ...MiddlewareOption) grpc.UnaryServerInterceptor {
	o := evaluateMiddlewareOptions(opts)
	return loggingInterceptor{logger: logger, middlewareOptions: o}.intercept
}

// StreamLoggingInterceptor provides a request-scoped log entry into context for
// Streaming gRPC requests, and logs request details at the end of the stream.
// Logging interceptors should be chained at the very top of the request scope.
func StreamLoggingInterceptor(logger *logrus.Logger, opts ...MiddlewareOption) grpc.StreamServerInterceptor {
	o := evaluateMiddlewareOptions(opts)
	return loggingInterceptor{logger: logger, middlewareOptions: o}.interceptStream
}

type loggingInterceptor struct {
	logger *logrus.Logger
	*middlewareOptions
}

// GRPCRequest represents details of a gRPC request and response appended to a log.
type GRPCRequest struct {
	Method    string `json:"method,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	PeerAddr  string `json:"peer,omitempty"`
	Deadline  string `json:"deadline,omitempty"`
	Duration  string `json:"duration,omitempty"`
}

func (l loggingInterceptor) intercept(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	startTime := time.Now()
	ctx = ctxlogrus.ToContext(ctx, logrus.NewEntry(l.logger))

	request := l.requestFromContext(ctx, info.FullMethod)

	resp, err := handler(ctx, req)

	request.Duration = fmt.Sprintf("%.9fs", time.Since(startTime).Seconds())

	l.log(ctx, err, info.FullMethod)

	return resp, err
}

func (l loggingInterceptor) interceptStream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	startTime := time.Now()
	ctx := ctxlogrus.ToContext(ss.Context(), logrus.NewEntry(l.logger))

	request := l.requestFromContext(ctx, info.FullMethod)

	wrapped := grpc_middleware.WrapServerStream(ss)
	wrapped.WrappedContext = ctx

	err := handler(srv, wrapped)

	request.Duration = fmt.Sprintf("%.9fs", time.Since(startTime).Seconds())

	l.log(ctx, err, info.FullMethod)

	return err
}

// requestFromContext creates gRPC request details with information extracted from the request context
func (l *loggingInterceptor) requestFromContext(ctx context.Context, method string) *GRPCRequest {
	request := &GRPCRequest{Method: method}

	if d, ok := ctx.Deadline(); ok {
		request.Deadline = d.UTC().Format(time.RFC3339Nano)
	}

	if p, ok := peer.FromContext(ctx); ok && p != nil {
		u := &url.URL{
			Scheme: p.Addr.Network(),
			Host:   p.Addr.String(),
		}
		request.PeerAddr = u.String()
	}

	if md, ok := metadata.FromIncomingContext(ctx); ok && md != nil {
		request.UserAgent = strings.Join(md.Get("user-agent"), "")

		if trace := md.Get("X-Cloud-Trace-Context"); len(trace) == 1 {
			ctxlogrus.AddFields(ctx, logrus.Fields{"trace": trace[0]})
		}
	}

	ctxlogrus.AddFields(ctx, logrus.Fields{"grpcRequest": request})

	return request
}

// logStatus adds the gRPC Status to the log context.
// If the response is an internal server error, log that as an Error
// returns true if the logging was handled (e.g. internal server error)
func (l *loggingInterceptor) log(ctx context.Context, err error, method string) {
	if !l.filterRPC(ctx, method, err) {
		return
	}

	if err != nil {
		// add grpcStatus to log entry, if available
		st := status.Convert(err)
		jsonStatus, merr := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(st.Proto())
		if merr == nil {
			ctxlogrus.AddFields(ctx, logrus.Fields{
				"grpcStatus": json.RawMessage(jsonStatus),
			})
			if st.Code() == codes.Internal {
				// if we're about to return an internal server error to the client, always log as Error level.
				// skip errorHandler
				ctxlogrus.Extract(ctx).WithError(err).Errorf("internal error response on RPC %s", method)
				return
			}
		} else {
			// this should never actually happen, so we log it to help identify
			// why our gRPC status error isn't included in logs
			ctxlogrus.Extract(ctx).WithError(merr).Warnf("error marshalling error status into log")
		}

		// opportunity to log or transform the error with a custom error handler
		// If the error handler indicates logging has been handled already, we
		// return early and do not log as Info down below
		if handled := l.errorHandler(ctx, err, method); handled {
			return
		}

	}

	// if we reach here, the response either wasn't a bad error worth handling (e.g. NotFound and its ilk)
	ctxlogrus.Extract(ctx).Infof("served RPC %v", method)
}

// RecoveryMiddleware recovers from panics in the HTTP handler chain, logging
// an error for Error Reporting
func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			var err error
			e := recover()
			if e == nil {
				return
			}

			switch t := e.(type) {
			case string:
				err = errors.New(t)
			case error:
				err = t
			default:
				err = fmt.Errorf("unknown panic value: (%T) %v", t, t)
			}

			ctx := r.Context()

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)

			stErr := errWithStack(ctx, err)
			entry := ctxlogrus.Extract(ctx)

			// write error back to client
			jsonStatus, merr := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(stErr.Proto())
			if merr != nil {
				entry.WithError(merr).Errorf("error marshalling error status into log")
				fmt.Fprint(w, `{"error": "server_error"}`)
				return
			}

			if _, err := w.Write(jsonStatus); err != nil {
				entry.WithError(err).Warn("error writing json server_error to ResponseWriter")
				return
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// UnaryRecoveryInterceptor is an interceptor that recovers panics and turns them
// into nicer GRPC errors.
func UnaryRecoveryInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
	defer func() {
		e := recover()
		if e == nil {
			return
		}

		switch t := e.(type) {
		case string:
			err = errors.New(t)
		case error:
			err = t
		default:
			err = fmt.Errorf("unknown panic value: (%T) %v", t, t)
		}

		stErr := errWithStack(ctx, err)
		err = stErr.Err()
		resp = nil
	}()

	return handler(ctx, req)
}

// StreamRecoveryInterceptor is an interceptor that recovers panics from
// Streaming services and turns them into nicer gRPC errors
func StreamRecoveryInterceptor(srv interface{}, ss grpc.ServerStream, _ *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
	defer func() {
		e := recover()
		if e == nil {
			return
		}

		switch t := e.(type) {
		case string:
			err = errors.New(t)
		case error:
			err = t
		default:
			err = fmt.Errorf("unknown panic value: (%T) %v", t, t)
		}

		stErr := errWithStack(ss.Context(), err)
		err = stErr.Err()
	}()

	return handler(srv, ss)
}

// errWithStack generates a stack trace, logs it, and provides an internal
// server error response back to return to the client
func errWithStack(ctx context.Context, err error) *status.Status {
	stack := debug.Stack()
	ctxlogrus.Extract(ctx).WithError(err).WithField("stackTrace", string(stack)).Error("panic handling request")

	serverError := status.New(codes.Internal, "server error")
	// generate a shared UUID we can find this log entry from client-provided response body
	stErr, _ := serverError.WithDetails(&errdetails.RequestInfo{
		RequestId: shortuuid.New(),
	})
	return stErr
}

// getRemoteIP extracts the remote IP from X-Forwarded-For header, if applicable
// https://cloud.google.com/load-balancing/docs/https#x-forwarded-for_header
func getRemoteIP(r *http.Request) string {
	ctx := r.Context()

	fwdHeader := r.Header.Get("X-Forwarded-For")
	forwarded := strings.Split(fwdHeader, ",")
	ctxlogrus.AddFields(ctx, logrus.Fields{
		"forwardIP": fwdHeader,
	})

	// x-Forwarded-For directly from GCP LB is assumed to be sanitized
	// format: `<unverified IP(s)>, <client IP>, <global fw rule ext. IP>, <other proxies IP>`
	// only second and third entries are added for requests through GCP
	if len(forwarded) >= 2 {
		return strings.TrimSpace(forwarded[len(forwarded)-2])
	}

	// fallback to peer IP
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "0.0.0.0"
	}
	return ip
}
