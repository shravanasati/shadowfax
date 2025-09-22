package server

import (
	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
)

// Represents a path handler function. Takes a request and returns a response.
type Handler func(*request.Request) response.Response
