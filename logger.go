package middleware

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

// Logger is an interface that allows users to implement a custom HTTP request
// logging algorithm. Logging, in the context of Internet web servers, is the
// act of keeping a history of page requests.
//
// Although there is a standard format for web server log files, usually known
// as Common Log Format, other proprietary formats exist. Information about the
// request, including client IP address, request date/time, page requested, the
// HTTP status code, bytes served, user agent, and referrer are typically added.
// This data can be combined into a single document, or separated into distinct
// logs, such as an access log, error log, or referrer log.
type Logger interface {
	// ListeningOn is called once, just before the execution of ListenAndServe.
	ListeningOn(string)
	// Shutdown is called once, immediately after the graceful server shutdown.
	Shutdown(error)
	// Log is called every time the web server handles a request.
	//
	// Below is an example of how to implement request tracing using Prometheus:
	//
	//   var router = middleware.New()
	//   var counter = prometheus.NewCounterVec(...)
	//   func init() {
	//       router.Logger = &Logger{}
	//       prometheus.MustRegister(counter)
	//       router.Handle(http.MethodGet, "/metrics", promhttp.Handler())
	//   }
	//   type Logger struct {}
	//   func (b *Logger) Log(data middleware.AccessLog) {
	//       counter.With(prometheus.Labels{"host": data.Host}).Inc()
	//   }
	//
	// If you want to extend the basic logger instead of override it, you can
	// embed it into your own using a private struct field, then use that field
	// to call the corresponding methods of the parent struct.
	//
	//   type Logger struct {
	//       parent middleware.Logger
	//   }
	//   func NewLogger() middleware.Logger {
	//       return &Logger{parent: middleware.NewBasicLogger()}
	//   }
	//   func (b *Logger) Log(data middleware.AccessLog) {
	//       // ... custom logging algorithm.
	//       b.parent.Log(data)
	//   }
	Log(AccessLog)
}

// AccessLog represents the most relevant information associated to each HTTP
// request. The struct type was designed to be as flexible as the Common Log
// Format, also known as the NCSA Common log format or simply NCSA_HTTPd, which
// is a standardized text file format used by various web servers like Apache
// and Nginx when generating server log files.
//
// Each line in a file stored in Common Log Format has the following syntax:
//
//   host ident authuser date request status bytes
//
// The format is extended by the Combined Log Format with the HTTP referrer and
// user-agent fields. The Logger interface gives you the flexibility to follow
// any standard or to design your own.
type AccessLog struct {
	StartTime     time.Time
	Host          string
	RemoteAddr    string
	RemoteUser    string
	Method        string
	Path          string
	Query         url.Values
	Protocol      string
	StatusCode    int
	BytesReceived int64
	BytesSent     int
	Header        http.Header
	Duration      time.Duration
}

// EmptyLogger implements the Logger interface to discard access logs.
type EmptyLogger struct{}

// ListeningOn implements the ListeningOn method for the Logger interface.
func (l *EmptyLogger) ListeningOn(addr string) {}

// Shutdown implements the Shutdown method for the Logger interface.
func (l *EmptyLogger) Shutdown(err error) {}

// Log implements the Log method for the Logger interface.
func (l *EmptyLogger) Log(data AccessLog) {}

// BasicLogger implements the Logger interface and the NCSA_HTTPd log format.
type BasicLogger struct {
	logger *log.Logger
}

// NewBasicLogger returns a new instance of a basic server access logger.
func NewBasicLogger() Logger {
	return &BasicLogger{
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

// ListeningOn implements the ListeningOn method for the Logger interface.
func (l *BasicLogger) ListeningOn(addr string) {
	l.logger.Println("listening on", addr)
}

// Shutdown implements the Shutdown method for the Logger interface.
func (l *BasicLogger) Shutdown(err error) {
	if err != nil {
		l.logger.Fatalf("server closed (err=%s)", err)
		return
	}

	l.logger.Println("server closed", "(ok)")
}

// Log implements the Log method for the Logger interface.
func (l *BasicLogger) Log(data AccessLog) {
	fullURL := data.Path

	if params := data.Query.Encode(); params != "" {
		fullURL += "?" + params
	}

	l.logger.Printf(
		"%s %s \"%s %s %s\" %d %d \"%s\" %v",
		data.Host,
		data.RemoteAddr,
		data.Method,
		fullURL,
		data.Protocol,
		data.StatusCode,
		data.BytesSent,
		data.Header.Get("User-Agent"),
		data.Duration,
	)
}
