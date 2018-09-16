package middleware

// AllowAccessExcept returns a "403 Forbidden" if the IP is blacklisted.
func (m *Middleware) AllowAccessExcept(ips []string) {
	m.restrictionType = allowAccessExcept
	m.deniedAddresses = ips
}

// DenyAccessExcept returns a "403 Forbidden" unless the IP is whitelisted.
func (m *Middleware) DenyAccessExcept(ips []string) {
	m.restrictionType = denyAccessExcept
	m.allowedAddresses = ips
}
