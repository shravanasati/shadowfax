package server

import (
	"github.com/shravanasati/shadowfax/internal/request"
	"github.com/shravanasati/shadowfax/internal/response"
)

type Handler func(*request.Request) response.Response
