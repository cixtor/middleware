package middleware

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
)

// startServer setups and starts the web server.
func (m *Middleware) startServer(address string, f func() error) error {
	addr, err := m.resolveTCPAddr(address)

	if err != nil {
		return err
	}

	m.serverInstance = &http.Server{
		Addr:              addr.String(),
		Handler:           m,
		ReadTimeout:       m.ReadTimeout,
		ReadHeaderTimeout: m.ReadHeaderTimeout,
		WriteTimeout:      m.WriteTimeout,
		IdleTimeout:       m.IdleTimeout,
		ErrorLog:          m.ErrorLog,
	}

	// Configure additional shutdown operations.
	m.serverInstance.RegisterOnShutdown(m.OnShutdown)

	m.Logger.ListeningOn(addr)

	err = f() /* ListenAndServe OR ListenAndServeTLS */

	// Ignore "http: Server closed" errors as benign.
	if err != nil && errors.Is(err, http.ErrServerClosed) {
		m.Logger.Shutdown(nil)
		return nil
	}

	m.Logger.Shutdown(err)

	return err
}

// resolveTCPAddr returns an address of TCP end point.
func (m *Middleware) resolveTCPAddr(address string) (net.Addr, error) {
	addr, err := net.ResolveTCPAddr("tcp", address)

	if err != nil {
		return &net.TCPAddr{}, err
	}

	l, err := net.ListenTCP("tcp", addr)

	if err != nil {
		return &net.TCPAddr{}, err
	}

	defer l.Close()

	return l.Addr(), nil
}

// FreePort returns a free TCP port from the local machine.
//
// An ephemeral port is a communications endpoint (port) of a transport layer
// protocol of the Internet protocol suite that is used for only a short period
// of time for the duration of a communication session. Such short-lived ports
// are allocated automatically within a predefined range of port numbers by the
// IP stack software of a computer operating system. The Transmission Control
// Protocol (TCP), the User Datagram Protocol (UDP), and the Stream Control
// Transmission Protocol (SCTP) typically use an ephemeral port for the
// client-end of a client–server communication.
//
// The allocation of an ephemeral port is temporary and only valid for the
// duration of the communication session. After completion of the session, the
// port is destroyed and the port number becomes available for reuse, but many
// implementations simply increment the last used port number until the
// ephemeral port range is exhausted, when the numbers roll over
//
// The RFC 6056 says that the range for ephemeral ports should be 1024-65535.
//
// Source: https://en.wikipedia.org/wiki/Ephemeral_port
func (m *Middleware) FreePort() (net.Addr, error) {
	return m.resolveTCPAddr("127.0.0.1:0")
}

// ListenAndServe listens on a TCP network address and then calls server.Serve
// to handle requests on incoming connections. All accepted connections are
// configured to enable TCP keep-alives. If the hostname is blank, ":http" is
// used. The method always returns a non-nil error.
func (m *Middleware) ListenAndServe(address string) error {
	return m.startServer(address, func() error {
		return m.serverInstance.ListenAndServe()
	})
}

// ListenAndServeTLS acts identically to ListenAndServe, except that it
// expects HTTPS connections. Additionally, files containing a certificate and
// matching private key for the server must be provided. If the certificate
// is signed by a certificate authority, the certFile should be the concatenation
// of the server's certificate, any intermediates, and the CA's certificate.
func (m *Middleware) ListenAndServeTLS(address string, certFile string, keyFile string, cfg *tls.Config) error {
	return m.startServer(address, func() error {
		m.serverInstance.TLSConfig = cfg /* TLS configuration */
		return m.serverInstance.ListenAndServeTLS(certFile, keyFile)
	})
}

// Shutdown gracefully shuts down the server without interrupting any active
// connections. Shutdown works by first closing all open listeners, then
// closing all idle connections, and then waiting indefinitely for connections
// to return to idle and then shut down.
//
// If the provided context expires before the shutdown is complete, Shutdown
// returns the context's error, otherwise it returns any error returned from
// closing the Server's underlying Listener(s).
func (m *Middleware) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), m.ShutdownTimeout)

	defer cancel()

	if m.serverInstance == nil {
		// Nothing to stop.
		return nil
	}

	return m.serverInstance.Shutdown(ctx)
}
