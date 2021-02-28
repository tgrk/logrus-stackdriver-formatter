package logadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// ProtoJSONMarshalOptions defines how we want to serialize request/response/status messages to logs
// can be overridden
var ProtoJSONMarshalOptions = protojson.MarshalOptions{
	UseEnumNumbers:  false,
	EmitUnpopulated: true,
}

var serverError = status.New(codes.Internal, "server error")

// LoggingMiddleware is a middleware for writing request logs in a structured
// format to stackdriver.
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
				RemoteIP:      r.RemoteAddr,
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

func UnaryLoggingInterceptor(logger *logrus.Logger, opts ...MiddlewareOption) grpc.UnaryServerInterceptor {
	o := evaluateMiddlewareOptions(opts)
	return loggingInterceptor{logger: logger, middlewareOptions: o}.intercept
}

func StreamLoggingInterceptor(logger *logrus.Logger, opts ...MiddlewareOption) grpc.StreamServerInterceptor {
	o := evaluateMiddlewareOptions(opts)
	return loggingInterceptor{logger: logger, middlewareOptions: o}.interceptStream
}

type loggingInterceptor struct {
	logger *logrus.Logger
	*middlewareOptions
}

type GRPCRequest struct {
	Method    string `json:"method,omitempty"`
	UserAgent string `json:"userAgent,omitempty"`
	PeerAddr  string `json:"peer,omitempty"`
	Deadline  string `json:"deadline,omitempty"`
	Duration  string `json:"duration,omitempty"`
}

func (l loggingInterceptor) intercept(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	startTime := time.Now()
	ctx = ctxlogrus.ToContext(ctx, logrus.NewEntry(l.logger))

	request := l.requestFromContext(ctx, &GRPCRequest{
		Method: info.FullMethod,
	})

	resp, err = handler(ctx, req)

	request.Duration = fmt.Sprintf("%.9fs", time.Since(startTime).Seconds())

	l.log(ctx, err, info.FullMethod)

	return
}

func (l loggingInterceptor) interceptStream(
	srv interface{},
	ss grpc.ServerStream,
	info *grpc.StreamServerInfo,
	handler grpc.StreamHandler,
) (err error) {
	startTime := time.Now()
	ctx := ctxlogrus.ToContext(ss.Context(), logrus.NewEntry(l.logger))

	request := l.requestFromContext(ctx, &GRPCRequest{
		Method: info.FullMethod,
	})

	wrapped := grpc_middleware.WrapServerStream(ss)
	wrapped.WrappedContext = ctx

	err = handler(srv, wrapped)

	request.Duration = fmt.Sprintf("%.9fs", time.Since(startTime).Seconds())

	l.log(ctx, err, info.FullMethod)

	return
}

func (l *loggingInterceptor) requestFromContext(ctx context.Context, request *GRPCRequest) *GRPCRequest {
	if d, ok := ctx.Deadline(); ok {
		request.Deadline = d.Format(time.RFC3339Nano)
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
		st := status.Convert(err)
		jsonStatus, merr := ProtoJSONMarshalOptions.Marshal(st.Proto())
		if merr != nil {
			ctxlogrus.Extract(ctx).WithError(merr).Errorf("error marshalling error status into log")
		}
		ctxlogrus.AddFields(ctx, logrus.Fields{
			"grpcStatus": json.RawMessage(jsonStatus),
		})

		if st.Code() == codes.Internal {
			ctxlogrus.Extract(ctx).WithError(err).Errorf("internal error response on RPC %s", method)
			return
		}
	}

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
				err = fmt.Errorf("unknown error: %w", t)
			}

			ctx := r.Context()

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)

			stErr := errWithStack(ctx, err)
			entry := ctxlogrus.Extract(ctx)

			// write error back to client
			jsonStatus, merr := ProtoJSONMarshalOptions.Marshal(stErr.Proto())
			if merr != nil {
				entry.WithError(merr).Errorf("error marshalling error status into log")
				fmt.Fprint(w, `{"error": "server_error"}`)
				return
			}

			if _, err := w.Write(jsonStatus); err != nil {
				entry.WithError(err).Fatal("error encoding json server_error to ResponseWriter")
				fmt.Fprint(w, `{"error": "server_error"}`)
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
			err = fmt.Errorf("unknown error: %w", t)
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
			err = fmt.Errorf("unknown error: %w", t)
		}

		stErr := errWithStack(ss.Context(), err)
		err = stErr.Err()
	}()

	return handler(srv, ss)
}

func errWithStack(ctx context.Context, err error) *status.Status {
	stack := debug.Stack()
	ctxlogrus.Extract(ctx).WithError(err).WithField("stackTrace", string(stack)).Error("panic handling request")

	// generate a shared UUID we can find this log entry from client-provided response body
	stErr, _ := serverError.WithDetails(&errdetails.RequestInfo{
		RequestId: shortuuid.New(),
	})
	return stErr
}
