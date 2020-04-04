package middleware

// AllowAccessExcept returns a "403 Forbidden" if the IP is blacklisted.
func (m *Middleware) AllowAccessExcept(ips []string) {
	m.restrictionType = allowAccessExcept
	m.deniedAddresses = ips
}
