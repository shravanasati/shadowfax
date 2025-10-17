package response

// RedirectResponse is a response that is used for redirection.
type RedirectResponse struct {
	Response
}

// NewRedirectResponse creates a new Redirect response. Uses status code 302 (Found) by default.
func NewRedirectResponse(location string) Response {
	br := NewBaseResponse().
		WithStatusCode(StatusFound).
		WithHeader("content-length", "0").
		WithHeader("location", location)

	return &RedirectResponse{Response: br}
}
