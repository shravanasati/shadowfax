package server

import (
	"log"
	"runtime/debug"
	"time"

	"github.com/shravanasati/shadowfax/internal/response"
)

// Server configuration options.
// Address defaults to `:42069`.
// Recovery function by default prints the stack trace and writes a 500 Internal Server Error response.
// Read and write timeout default to 0, implying there's no timeout on either operation.
type ServerOpts struct {
	// The address for the server to listen on.
	Address string

	// Recovery function takes the return value of the recover() call as input and returns a response that is written to the connection. The connection is closed after writing the response.
	Recovery func(any) response.Response

	// Sets a read deadline on the underlying connection.
	ReadTimeout time.Duration

	// Sets a write deadline on the underlying connection.
	WriteTimeout time.Duration
}

var defaultRecovery = func(r any) response.Response {
	log.Println("recovered from panic:", r)
	debug.PrintStack()

	errorStatusCode := response.StatusInternalServerError
	resp := response.
		NewTextResponse(response.GetStatusReason(errorStatusCode)).
		WithStatusCode(errorStatusCode)
	return resp
}
