package server

import (
	"io"

	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
)

type HandlerError struct {
	statusCode response.StatusCode
	message    string
}

func NewHandlerError(statusCode response.StatusCode, messsage string) *HandlerError {
	return &HandlerError{statusCode: statusCode, message: messsage}
}

func (he *HandlerError) response() *response.Response {
	return response.NewResponse().WithStatusCode(he.statusCode).WithBodyString(he.message)
}

type Handler func(io.Writer, *request.Request) *HandlerError
