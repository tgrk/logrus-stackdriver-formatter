package logadapter

type StackTraceStyle int

const (
	TraceInMessage StackTraceStyle = iota
	TraceInPayload
	TraceInBoth
)

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

// WithSourceReference adds reference to the source code.
func WithSourceReference(repository, revision string) Option {
	return func(f *Formatter) {
		f.SourceReference = append(f.SourceReference, SourceReference{
			Repository: repository,
			RevisionID: revision,
		})
	}
}

// WithProjectID makes sure all entries have your Project information.
func WithProjectID(i string) Option {
	return func(f *Formatter) {
		f.ProjectID = i
	}
}

// WithStackSkip lets you configure which packages should be skipped for locating the error.
func WithStackSkip(v string) Option {
	return func(f *Formatter) {
		f.StackSkip = append(f.StackSkip, v)
	}
}

// WithRegexSkip lets you configure
// which functions or packages should be skipped for locating the error.
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

// WithStackTraceStyle configures where to write the stacktrace:
// appended to the message, as its own field, or both
func WithStackTraceStyle(s StackTraceStyle) Option {
	return func(f *Formatter) {
		f.StackStyle = s
	}
}

// WithPrettyPrint pretty-prints logs.
func WithPrettyPrint() Option {
	return func(f *Formatter) {
		f.PrettyPrint = true
	}
}
