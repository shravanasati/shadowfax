package middleware

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/shravanasati/shadowfax/request"
	"github.com/shravanasati/shadowfax/response"
	"github.com/shravanasati/shadowfax/server"
)

var safeMethods = []string{"GET", "HEAD", "OPTIONS", "TRACE"}
var defaultDenyHandler server.Handler = func(_ *request.Request) response.Response {
	return response.NewBaseResponse().WithStatusCode(response.StatusForbidden)
}

func validateOrigin(o string) error {
	u, err := url.Parse(o)
	if err != nil {
		return fmt.Errorf("invalid origin %q: %w", o, err)
	}
	if u.Scheme == "" {
		return fmt.Errorf("invalid origin %q: scheme is required", o)
	}
	if u.Host == "" {
		return fmt.Errorf("invalid origin %q: host is required", o)
	}
	if u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("invalid origin %q: path, query, and fragment are not allowed", o)
	}

	return nil
}

// CORF protects server against Cross-Origin Request Forgery. Use NewCORF and CORF.Handler.
type CORF struct {
	trustedMu      sync.RWMutex
	trustedOrigins map[string]bool
	deny           atomic.Pointer[server.Handler] // if nil, falls back to defaultDenyHandler
}

// NewCORF constructs a CORF instance and validates initial trusted origins.
// CORFMiddleware protects server against the Cross-Origin Request Forgery attacks.
// Most of the code is inspired by go's http.CrossOriginProtection middleware.
// Errors are returned when trusted origins are not valid origins.
// If the request is not valid, the deny handler is called.
func NewCORF(trustedOrigins ...string) (*CORF, error) {
	c := &CORF{trustedOrigins: make(map[string]bool)}
	for _, or := range trustedOrigins {
		if err := validateOrigin(or); err != nil {
			return nil, err
		}
		c.trustedOrigins[or] = true
	}
	return c, nil
}

// AddTrustedOrigin adds a trusted origin to this CORF instance.
func (c *CORF) AddTrustedOrigin(origin string) error {
	if err := validateOrigin(origin); err != nil {
		return err
	}
	c.trustedMu.Lock()
	if c.trustedOrigins == nil {
		c.trustedOrigins = make(map[string]bool)
	}
	c.trustedOrigins[origin] = true
	c.trustedMu.Unlock()
	return nil
}

// SetDenyHandler sets a per-instance deny handler; pass nil to use default.
func (c *CORF) SetDenyHandler(h server.Handler) {
	if h == nil {
		var nilPtr *server.Handler
		c.deny.Store(nilPtr)
		return
	}
	c.deny.Store(&h)
}

func (c *CORF) effectiveDeny() server.Handler {
	if p := c.deny.Load(); p != nil {
		return *p
	}
	return defaultDenyHandler
}

// Handler returns a middleware-wrapped handler that enforces CORF rules.
func (c *CORF) Handler(next server.Handler) server.Handler {
	return func(r *request.Request) response.Response {
		if slices.Contains(safeMethods, r.Method) {
			// allow requests if they are safe methods
			return next(r)
		}

		origin := r.Headers.Get("Origin")
		originPresent := len(origin) != 0
		// read trusted origins under RLock
		c.trustedMu.RLock()
		trusted := origin != "" && c.trustedOrigins[origin]
		c.trustedMu.RUnlock()
		if trusted {
			// allow requests if they are from a trusted origin
			return next(r)
		}

		secFetchSite := strings.ToLower(r.Headers.Get("Sec-Fetch-Site"))
		secFetchSitePresent := len(secFetchSite) != 0
		if secFetchSitePresent {
			if secFetchSite == "same-origin" || secFetchSite == "none" {
				return next(r)
			}
			return c.effectiveDeny()(r)
		}

		if !originPresent {
			return next(r)
		}

		host := r.Headers.Get("Host")
		if o, err := url.Parse(origin); err == nil && o.Host == host {
			// origin matches host
			return next(r)
		}

		return c.effectiveDeny()(r)
	}
}
