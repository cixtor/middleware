package middleware

// folder represents a single portion of an endpoint.
//
// In the following example, there are six (6) folders.
//
//	┌───────────────────────────────────┐
//	│    /usr/local/etc/openssl/cert.pem│
//	└───────────────────────────────────┘
//	┌────┬───┬─────┬───┬───────┬────────┐
//	│root│usr│local│etc│openssl│cert.pem│
//	└──┬─┴─┬─┴──┬──┴─┬─┴───┬───┴────┬───┘
//	   0   1    2    3     4        5
type folder string

// Name returns the identifier for the dynamic parameter.
//
// Note: Do not call before checking for s.IsParam() first.
func (s folder) Name() string {
	return string(s)[1:]
}

// IsParam determines whether the folder represents a dynamic name or not.
//
//	┌────────────────────────────────────┐
//	│    /usr/local/:group/:user/cert.pem│
//	└────────────────────────────────────┘
//	┌────┬───┬─────┬──────┬─────┬────────┐
//	│root│usr│local│:group│:user│cert.pem│
//	└──┬─┴─┬─┴──┬──┴───┬──┴──┬──┴────┬───┘
//	   F   F    F    True   True     F
func (s folder) IsParam() bool {
	return len(s) > 0 && s[0] == ':'
}

// IsGlob determines if the folder matches everything on the right.
//
//	┌───────────────────────────────┐
//	│    /usr/local/:group/:user/*  │
//	└───────────────────────────────┘
//	┌────┬───┬─────┬──────┬─────┬───┐
//	│root│usr│local│:group│:user│ * │
//	└──┬─┴─┬─┴──┬──┴───┬──┴──┬──┴─┬─┘
//	   F   F    F      F     F   True
func (s folder) IsGlob() bool {
	return s == "*"
}
