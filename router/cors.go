package router

import (
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/shravanasati/shadowfax/headers"
	"github.com/shravanasati/shadowfax/request"
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

// corsHandler handles preflight and actual requests.
type corsHandler struct {
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

func newCorsHandler(options CorsOptions) *corsHandler {
	c := &corsHandler{
		exposedHeaders:    convert(options.ExposedHeaders, http.CanonicalHeaderKey),
		allowOriginFunc:   options.AllowOriginFunc,
		allowCredentials:  options.AllowCredentials,
		maxAge:            options.MaxAge,
		optionPassthrough: options.OptionsPassthrough,
	}

	if len(options.AllowedOrigins) == 0 {
		if options.AllowOriginFunc == nil {
			c.allowedOriginsAll = true
		}
	} else {
		c.allowedOrigins = []string{}
		c.allowedWOrigins = []wildcard{}
		for _, origin := range options.AllowedOrigins {
			origin = strings.ToLower(origin)
			if origin == "*" {
				c.allowedOriginsAll = true
				c.allowedOrigins = nil
				c.allowedWOrigins = nil
				break
			} else if i := strings.IndexByte(origin, '*'); i >= 0 {
				w := wildcard{origin[0:i], origin[i+1:]}
				c.allowedWOrigins = append(c.allowedWOrigins, w)
			} else {
				c.allowedOrigins = append(c.allowedOrigins, origin)
			}
		}
	}

	if len(options.AllowedHeaders) == 0 {
		c.allowedHeaders = []string{"Origin", "Accept", "Content-Type"}
	} else {
		c.allowedHeaders = convert(append(options.AllowedHeaders, "Origin"), http.CanonicalHeaderKey)
		if slices.Contains(options.AllowedHeaders, "*") {
			c.allowedHeadersAll = true
			c.allowedHeaders = nil
		}
	}

	if len(options.AllowedMethods) == 0 {
		c.allowedMethods = []string{"GET", "POST", "HEAD"}
	} else {
		c.allowedMethods = convert(options.AllowedMethods, strings.ToUpper)
	}

	return c
}

func (c *corsHandler) handlePreflight(r *request.Request) *headers.Headers {
	headers := headers.NewHeaders()
	origin := r.Headers.Get("Origin")

	if r.Method != string(request.OPTIONS) {
		return headers
	}

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
	headers.Set("Access-Control-Allow-Methods", strings.ToUpper(reqMethod))
	if len(reqHeaders) > 0 {
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

func (c *corsHandler) handleActualRequest(r *request.Request) *headers.Headers {
	headers := headers.NewHeaders()
	origin := r.Headers.Get("Origin")
	hasOriginHeader := len(origin) != 0

	headers.Add("Vary", "Origin")

	if !hasOriginHeader {
		return headers
	}
	if !c.isOriginAllowed(r, origin) {
		return headers
	}

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

func (c *corsHandler) isOriginAllowed(r *request.Request, origin string) bool {
	if c.allowOriginFunc != nil {
		return c.allowOriginFunc(r, origin)
	}
	if c.allowedOriginsAll {
		return true
	}
	origin = strings.ToLower(origin)
	if slices.Contains(c.allowedOrigins, origin) {
			return true
		}
	for _, w := range c.allowedWOrigins {
		if w.match(origin) {
			return true
		}
	}
	return false
}

func (c *corsHandler) isMethodAllowed(method string) bool {
	method = strings.ToUpper(method)
	if method == string(request.OPTIONS) {
		return true
	}
	if len(c.allowedMethods) == 0 {
		return false
	}
	return slices.Contains(c.allowedMethods, method)
}

func (c *corsHandler) areHeadersAllowed(requestedHeaders []string) bool {
	if c.allowedHeadersAll || len(requestedHeaders) == 0 {
		return true
	}
	for _, header := range requestedHeaders {
		header = http.CanonicalHeaderKey(header)
		found := slices.Contains(c.allowedHeaders, header)
		if !found {
			return false
		}
	}
	return true
}
