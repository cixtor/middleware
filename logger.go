package middleware

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
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
//
// If you want to extend the basic logger instead of override it, you can
// embed it into your own using a private struct field, then use that field
// to call the corresponding methods of the parent struct.
//
//	type CustomLogger struct {
//	    parent middleware.Logger
//	}
//	func NewCustomLogger() middleware.Logger {
//	    return &CustomLogger{
//	        parent: middleware.NewBasicLogger(),
//	    }
//	}
//	func (l CustomLogger) Log(data middleware.AccessLog) {
//	    l.parent.Log(data)
//	}
//
// Example, request tracing using Prometheus:
//
//	var srv = middleware.New()
//	var counter = prometheus.NewCounterVec(...)
//	func init() {
//	    srv.Logger = NewCustomLogger()
//	    prometheus.MustRegister(counter)
//	    srv.Handle(http.MethodGet, "/metrics", promhttp.Handler())
//	}
//	type CustomLogger struct {
//	    parent middleware.Logger
//	}
//	func NewCustomLogger() middleware.Logger {
//	    return &CustomLogger{
//	        parent: middleware.NewBasicLogger(),
//	    }
//	}
//	func (l CustomLogger) Log(data middleware.AccessLog) {
//	    counter.With(prometheus.Labels{"host": data.Host}).Inc()
//	    l.parent.Log(data)
//	}
type Logger interface {
	// ListeningOn is called once, just before the execution of ListenAndServe.
	ListeningOn(net.Addr)
	// Shutdown is called once, immediately after the graceful server shutdown.
	Shutdown(error)
	// Log is called every time the web server handles a request.
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
//	host ident authuser date request status bytes
//
// Example:
//
//	127.0.0.1 - cixtor [10/Dec/2019:13:55:36 -0700] "GET /server-status HTTP/1.1" 200 2326
//
// The format is extended by the Combined Log Format with the HTTP referrer and
// user-agent fields. The Logger interface gives you the flexibility to follow
// any standard or to design your own.
//
// Example:
//
//	127.0.0.1 - cixtor [10/Dec/2019:13:55:36 -0700] "GET /server-status HTTP/1.1" 200 2326 "http://localhost/" "Mozilla/5.0 (KHTML, like Gecko) Version/78.0.3904.108"
//
// The "hyphen" in the output indicates that the requested piece of information
// is not available. In the example, the hyphen is the RFC 1413 identity of the
// client determined by "identd" on the client's machine. This information is
// highly unreliable and should almost never be used except on tightly controlled
// internal networks. Other web servers, like Apache httpd, will not even attempt
// to determine this information unless IdentityCheck is set to "On".
//
// Source: https://en.wikipedia.org/wiki/Common_Log_Format
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

// Request concatenates the request method, path, parameters and protocol.
func (a AccessLog) Request() string {
	return fmt.Sprintf("%q", a.Method+"\x20"+a.FullURL()+"\x20"+a.Protocol)
}

// FullURL concatenates the request path and its query parameters.
func (a AccessLog) FullURL() string {
	fullURL := a.Path

	if params := a.Query.Encode(); params != "" {
		fullURL += "?" + params
	}

	return fullURL
}

// Referer returns the request HTTP referer or a hyphen.
func (a AccessLog) Referer() string {
	referer := a.Header.Get("Referer")

	if referer == "" {
		referer = "-"
	}

	return referer
}

// UserAgent returns the request HTTP user-agent or a hyphen.
func (a AccessLog) UserAgent() string {
	userAgent := a.Header.Get("User-Agent")

	if userAgent == "" {
		userAgent = "-"
	}

	return userAgent
}

// String returns the request metadata in Combined Log format.
func (a AccessLog) String() string {
	return fmt.Sprintf(
		"%s %s %s %d %d %q %v",
		a.Host,
		a.RemoteAddr,
		a.Request(),
		a.StatusCode,
		a.BytesSent,
		a.Header.Get("User-Agent"),
		a.Duration,
	)
}

// CommonLog returns the request metadata in Common Log format.
func (a AccessLog) CommonLog() string {
	return fmt.Sprintf(
		"%s - - [%s] %s %d %d",
		a.RemoteAddr,
		a.StartTime.Format(`02/01/2006:15:04:05 -07:00`),
		a.Request(),
		a.StatusCode,
		a.BytesSent,
	)
}

// CombinedLog returns the request metadata in Combined Log format.
func (a AccessLog) CombinedLog() string {
	return fmt.Sprintf(
		"%s %q %q",
		a.CommonLog(),
		a.Referer(),
		a.UserAgent(),
	)
}
