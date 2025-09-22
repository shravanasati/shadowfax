package server

import (
	"log"
	"runtime/debug"
	"time"

	"github.com/shravanasati/shadowfax/internal/response"
)

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
	resp := response.NewTextResponse(response.GetStatusReason(response.StatusInternalServerError)).WithStatusCode(response.StatusInternalServerError)
	return resp
}
