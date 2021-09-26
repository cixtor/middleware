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
func (m *Middleware) FreePort() (net.Addr, error) {
	return m.resolveTCPAddr(":0")
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
