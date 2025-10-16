// Code taken from https://github.com/go-chi/cors/blob/master/cors.go

package cors

import (
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/shravanasati/shadowfax/headers"
	"github.com/shravanasati/shadowfax/request"
	"github.com/shravanasati/shadowfax/response"
	"github.com/shravanasati/shadowfax/server"
)

// CorsOptions is a configuration container to setup the CORS middleware.
type CorsOptions struct {
	// AllowedOrigins is a list of origins a cross-domain request can be executed from.
	// If the special "*" value is present in the list, all origins will be allowed.
	// An origin may contain a wildcard (*) to replace 0 or more characters
	// (i.e.: http://*.domain.com). Usage of wildcards implies a small performance penalty.
	// Only one wildcard can be used per origin.
	// Default value is ["*"]
	AllowedOrigins []string

	// AllowOriginFunc is a custom function to validate the origin. It takes the origin
	// as argument and returns true if allowed or false otherwise. If this option is
	// set, the content of AllowedOrigins is ignored.
	AllowOriginFunc func(r *request.Request, origin string) bool

	// AllowedMethods is a list of methods the client is allowed to use with
	// cross-domain requests. Default value is simple methods (HEAD, GET and POST).
	AllowedMethods []string

	// AllowedHeaders is list of non simple headers the client is allowed to use with
	// cross-domain requests.
	// If the special "*" value is present in the list, all headers will be allowed.
	// Default value is [] but "Origin" is always appended to the list.
	AllowedHeaders []string

	// ExposedHeaders indicates which headers are safe to expose to the API of a CORS
	// API specification
	ExposedHeaders []string

	// AllowCredentials indicates whether the request can include user credentials like
	// cookies, HTTP authentication or client side SSL certificates.
	AllowCredentials bool

	// MaxAge indicates how long (in seconds) the results of a preflight request
	// can be cached
	MaxAge int

	// OptionsPassthrough instructs preflight to let other potential next handlers to
	// process the OPTIONS method. Turn this on if your application handles OPTIONS.
	OptionsPassthrough bool
}

// CorsMiddleware http handler
type CorsMiddleware struct {
	// Normalized list of plain allowed origins
	allowedOrigins []string

	// List of allowed origins containing wildcards
	allowedWOrigins []wildcard

	// Optional origin validator function
	allowOriginFunc func(r *request.Request, origin string) bool

	// Normalized list of allowed headers
	allowedHeaders []string

	// Normalized list of allowed methods
	allowedMethods []string

	// Normalized list of exposed headers
	exposedHeaders []string
	maxAge         int

	// Set to true when allowed origins contains a "*"
	allowedOriginsAll bool

	// Set to true when allowed headers contains a "*"
	allowedHeadersAll bool

	allowCredentials  bool
	optionPassthrough bool
}

// NewCorsMiddleware creates a new Cors handler with the provided options.
func NewCorsMiddleware(options CorsOptions) *CorsMiddleware {
	c := &CorsMiddleware{
		exposedHeaders:    convert(options.ExposedHeaders, http.CanonicalHeaderKey),
		allowOriginFunc:   options.AllowOriginFunc,
		allowCredentials:  options.AllowCredentials,
		maxAge:            options.MaxAge,
		optionPassthrough: options.OptionsPassthrough,
	}

	// Normalize options
	// Note: for origins and methods matching, the spec requires a case-sensitive matching.
	// As it may error prone, we chose to ignore the spec here.

	// Allowed Origins
	if len(options.AllowedOrigins) == 0 {
		if options.AllowOriginFunc == nil {
			// Default is all origins
			c.allowedOriginsAll = true
		}
	} else {
		c.allowedOrigins = []string{}
		c.allowedWOrigins = []wildcard{}
		for _, origin := range options.AllowedOrigins {
			// Normalize
			origin = strings.ToLower(origin)
			if origin == "*" {
				// If "*" is present in the list, turn the whole list into a match all
				c.allowedOriginsAll = true
				c.allowedOrigins = nil
				c.allowedWOrigins = nil
				break
			} else if i := strings.IndexByte(origin, '*'); i >= 0 {
				// Split the origin in two: start and end string without the *
				w := wildcard{origin[0:i], origin[i+1:]}
				c.allowedWOrigins = append(c.allowedWOrigins, w)
			} else {
				c.allowedOrigins = append(c.allowedOrigins, origin)
			}
		}
	}

	// Allowed Headers
	if len(options.AllowedHeaders) == 0 {
		// Use sensible defaults
		c.allowedHeaders = []string{"Origin", "Accept", "Content-Type"}
	} else {
		// Origin is always appended as some browsers will always request for this header at preflight
		c.allowedHeaders = convert(append(options.AllowedHeaders, "Origin"), http.CanonicalHeaderKey)
		for _, h := range options.AllowedHeaders {
			if h == "*" {
				c.allowedHeadersAll = true
				c.allowedHeaders = nil
				break
			}
		}
	}

	// Allowed Methods
	if len(options.AllowedMethods) == 0 {
		// Default is spec's "simple" methods
		c.allowedMethods = []string{"GET", "POST", "HEAD"}
	} else {
		c.allowedMethods = convert(options.AllowedMethods, strings.ToUpper)
	}

	return c
}

// Handler creates a new Cors handler with passed options.
func Handler(options CorsOptions) func(next server.Handler) server.Handler {
	c := NewCorsMiddleware(options)
	return c.Handler
}

// AllowAll create a new Cors handler with permissive configuration allowing all
// origins with all standard methods with any header and credentials.
func AllowAll() *CorsMiddleware {
	return NewCorsMiddleware(CorsOptions{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{
			string(request.DELETE),
			string(request.HEAD),
			string(request.GET),
			string(request.POST),
			string(request.PUT),
			string(request.PATCH),
		},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
	})
}

// Handler apply the CORS specification on the request, and add relevant CORS headers
// as necessary.
func (c *CorsMiddleware) Handler(next server.Handler) server.Handler {
	return server.Handler(func(r *request.Request) response.Response {
		// null or empty Origin header value is acceptable and it is considered having that header
		origin := r.Headers.Get("Origin")
		hasOriginHeader := len(origin) != 0

		if r.Method == string(request.OPTIONS) && r.Headers.Get("Access-Control-Request-Method") != "" && hasOriginHeader {
			headers := c.handlePreflight(r)
			// Preflight requests are standalone and should stop the chain as some other
			// middleware may not handle OPTIONS requests correctly. One typical example
			// is authentication middleware ; OPTIONS requests won't carry authentication
			// headers (see #1)
			resp := response.NewBaseResponse()
			if c.optionPassthrough {
				resp = next(r)
			} else {
				resp.WithStatusCode(response.StatusOK)
			}
			// Set CORS headers, using Set() to avoid duplication
			for k, v := range maps.Collect(headers.All()) {
				resp.GetHeaders().Set(k, v)
			}
			return resp
		} else {
			headers := c.handleActualRequest(r)
			res := next(r)
			res.WithHeaders(maps.Collect(headers.All()))
			return res
		}
	})
}

// handlePreflight handles pre-flight CORS requests
func (c *CorsMiddleware) handlePreflight(r *request.Request) *headers.Headers {
	headers := headers.NewHeaders()
	origin := r.Headers.Get("Origin")

	if r.Method != string(request.OPTIONS) {
		return headers
	}
	// Always set Vary headers
	// see https://github.com/rs/cors/issues/10,
	//     https://github.com/rs/cors/commit/dbdca4d95feaa7511a46e6f1efb3b3aa505bc43f#commitcomment-12352001
	headers.Add("Vary", "Origin")
	headers.Add("Vary", "Access-Control-Request-Method")
	headers.Add("Vary", "Access-Control-Request-Headers")

	if !c.isOriginAllowed(r, origin) {
		return headers
	}

	reqMethod := r.Headers.Get("Access-Control-Request-Method")
	if !c.isMethodAllowed(reqMethod) {
		return headers
	}
	reqHeaders := parseHeaderList(r.Headers.Get("Access-Control-Request-Headers"))
	if !c.areHeadersAllowed(reqHeaders) {
		return headers
	}
	if c.allowedOriginsAll {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Set("Access-Control-Allow-Origin", origin)
	}
	// Spec says: Since the list of methods can be unbounded, simply returning the method indicated
	// by Access-Control-Request-Method (if supported) can be enough
	headers.Set("Access-Control-Allow-Methods", strings.ToUpper(reqMethod))
	if len(reqHeaders) > 0 {
		// Spec says: Since the list of headers can be unbounded, simply returning supported headers
		// from Access-Control-Request-Headers can be enough
		headers.Set("Access-Control-Allow-Headers", strings.Join(reqHeaders, ", "))
	}
	if c.allowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}
	if c.maxAge > 0 {
		headers.Set("Access-Control-Max-Age", strconv.Itoa(c.maxAge))
	}

	return headers
}

// handleActualRequest handles simple cross-origin requests, actual request or redirects
func (c *CorsMiddleware) handleActualRequest(r *request.Request) *headers.Headers {
	headers := headers.NewHeaders()
	// null Origin header value is acceptable and it is considered having that header
	origin := r.Headers.Get("Origin")
	hasOriginHeader := len(origin) != 0

	// Always set Vary, see https://github.com/rs/cors/issues/10
	headers.Add("Vary", "Origin")

	if !hasOriginHeader {
		return headers
	}
	if !c.isOriginAllowed(r, origin) {
		return headers
	}

	// Note that spec does define a way to specifically disallow a simple method like GET or
	// POST. Access-Control-Allow-Methods is only used for pre-flight requests and the
	// spec doesn't instruct to check the allowed methods for simple cross-origin requests.
	// We think it's a nice feature to be able to have control on those methods though.
	if !c.isMethodAllowed(r.Method) {
		return headers
	}
	if c.allowedOriginsAll {
		headers.Set("Access-Control-Allow-Origin", "*")
	} else {
		headers.Set("Access-Control-Allow-Origin", origin)
	}
	if len(c.exposedHeaders) > 0 {
		headers.Set("Access-Control-Expose-Headers", strings.Join(c.exposedHeaders, ", "))
	}
	if c.allowCredentials {
		headers.Set("Access-Control-Allow-Credentials", "true")
	}

	return headers
}

// isOriginAllowed checks if a given origin is allowed to perform cross-domain requests
// on the endpoint
func (c *CorsMiddleware) isOriginAllowed(r *request.Request, origin string) bool {
	if c.allowOriginFunc != nil {
		return c.allowOriginFunc(r, origin)
	}
	if c.allowedOriginsAll {
		return true
	}
	origin = strings.ToLower(origin)
	for _, o := range c.allowedOrigins {
		if o == origin {
			return true
		}
	}
	for _, w := range c.allowedWOrigins {
		if w.match(origin) {
			return true
		}
	}
	return false
}

// isMethodAllowed checks if a given method can be used as part of a cross-domain request
// on the endpoint
func (c *CorsMiddleware) isMethodAllowed(method string) bool {
	if len(c.allowedMethods) == 0 {
		// If no method allowed, always return false, even for preflight request
		return false
	}
	method = strings.ToUpper(method)
	if method == string(request.OPTIONS) {
		// Always allow preflight requests
		return true
	}
	return slices.Contains(c.allowedMethods, method)
}

// areHeadersAllowed checks if a given list of headers are allowed to used within
// a cross-domain request.
func (c *CorsMiddleware) areHeadersAllowed(requestedHeaders []string) bool {
	if c.allowedHeadersAll || len(requestedHeaders) == 0 {
		return true
	}
	for _, header := range requestedHeaders {
		header = http.CanonicalHeaderKey(header)
		found := false
		for _, h := range c.allowedHeaders {
			if h == header {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
